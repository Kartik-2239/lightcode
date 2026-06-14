package oauth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Kartik-2239/lightcode/internal/server/config"
)

const codexOAuthClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
const codexLoginDefaultPort = 1455
const codexLoginFallbackPort = 1457

var codexAuthBaseURL = "https://auth.openai.com"

type codexUserCodeResponse struct {
	DeviceAuthID string `json:"device_auth_id"`
	UserCode     string `json:"user_code"`
	Usercode     string `json:"usercode"`
	IntervalRaw  any    `json:"interval"`
}

type codexTokenPollResponse struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeChallenge     string `json:"code_challenge"`
	CodeVerifier      string `json:"code_verifier"`
}

type CodexLoginFlow struct {
	AuthURL string
	Device  *DeviceCodeResponse
	Browser *CodexBrowserAuthFlow
}

type CodexBrowserAuthFlow struct {
	server       *http.Server
	listener     net.Listener
	redirectURI  string
	codeVerifier string
	state        string
	done         chan codexBrowserCallbackResult
	once         sync.Once
}

type codexBrowserCallbackResult struct {
	code string
	err  error
}

func StartCodexLoginFlow() (CodexLoginFlow, error) {
	browserFlow, err := StartCodexBrowserAuthFlow()
	if err == nil {
		return CodexLoginFlow{
			AuthURL: browserFlow.AuthURL(),
			Browser: browserFlow,
		}, nil
	}

	device, deviceErr := StartCodexAuthFlow()
	if deviceErr != nil {
		return CodexLoginFlow{}, fmt.Errorf("browser login failed: %v; device login failed: %w", err, deviceErr)
	}
	return CodexLoginFlow{
		AuthURL: device.VerificationURI,
		Device:  &device,
	}, nil
}

func WaitForCodexLogin(flow CodexLoginFlow) (config.AuthVal, error) {
	if flow.Browser != nil {
		return flow.Browser.Wait()
	}
	if flow.Device != nil {
		return WaitForCodexAuth(*flow.Device)
	}
	return config.AuthVal{}, fmt.Errorf("codex login flow was not started")
}

func StartCodexBrowserAuthFlow() (*CodexBrowserAuthFlow, error) {
	pkce, err := generateCodexPKCE()
	if err != nil {
		return nil, err
	}
	state, err := randomBase64URL(32)
	if err != nil {
		return nil, err
	}

	listener, err := listenCodexCallback()
	if err != nil {
		return nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	flow := &CodexBrowserAuthFlow{
		listener:     listener,
		redirectURI:  fmt.Sprintf("http://localhost:%d/auth/callback", port),
		codeVerifier: pkce.codeVerifier,
		state:        state,
		done:         make(chan codexBrowserCallbackResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", flow.handleCallback)
	mux.HandleFunc("/cancel", flow.handleCancel)
	flow.server = &http.Server{Handler: mux}
	go func() {
		if err := flow.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			flow.finish(codexBrowserCallbackResult{err: err})
		}
	}()

	return flow, nil
}

func (f *CodexBrowserAuthFlow) AuthURL() string {
	query := url.Values{}
	query.Set("response_type", "code")
	query.Set("client_id", codexOAuthClientID)
	query.Set("redirect_uri", f.redirectURI)
	query.Set("scope", "openid profile email offline_access api.connectors.read api.connectors.invoke")
	query.Set("code_challenge", codexCodeChallenge(f.codeVerifier))
	query.Set("code_challenge_method", "S256")
	query.Set("id_token_add_organizations", "true")
	query.Set("codex_cli_simplified_flow", "true")
	query.Set("state", f.state)
	query.Set("originator", "lightcode")
	return codexBaseURL() + "/oauth/authorize?" + query.Encode()
}

func (f *CodexBrowserAuthFlow) Wait() (config.AuthVal, error) {
	defer f.shutdown()
	select {
	case result := <-f.done:
		if result.err != nil {
			return config.AuthVal{}, result.err
		}
		return exchangeCodexCodeForTokens(result.code, f.codeVerifier, f.redirectURI)
	case <-time.After(15 * time.Minute):
		return config.AuthVal{}, fmt.Errorf("codex browser login expired")
	}
}

func (f *CodexBrowserAuthFlow) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("state") != f.state {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}
	if errorCode := r.URL.Query().Get("error"); errorCode != "" {
		description := r.URL.Query().Get("error_description")
		message := errorCode
		if strings.TrimSpace(description) != "" {
			message = description
		}
		http.Error(w, "Sign-in failed: "+message, http.StatusForbidden)
		f.finish(codexBrowserCallbackResult{err: fmt.Errorf("codex browser login failed: %s", message)})
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		f.finish(codexBrowserCallbackResult{err: fmt.Errorf("codex browser login did not return an authorization code")})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, "<!doctype html><title>Lightcode login complete</title><p>Login complete. You can close this tab and return to Lightcode.</p>")
	f.finish(codexBrowserCallbackResult{code: code})
}

func (f *CodexBrowserAuthFlow) handleCancel(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Login cancelled", http.StatusRequestTimeout)
	f.finish(codexBrowserCallbackResult{err: fmt.Errorf("codex browser login cancelled")})
}

func (f *CodexBrowserAuthFlow) finish(result codexBrowserCallbackResult) {
	f.once.Do(func() {
		f.done <- result
	})
}

func (f *CodexBrowserAuthFlow) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if f.server != nil {
		_ = f.server.Shutdown(ctx)
	}
	if f.listener != nil {
		_ = f.listener.Close()
	}
}

func StartCodexAuthFlow() (DeviceCodeResponse, error) {
	body, err := json.Marshal(map[string]string{"client_id": codexOAuthClientID})
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	req, err := http.NewRequest("POST", codexAccountsURL()+"/deviceauth/usercode", bytes.NewReader(body))
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "lightcode")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return DeviceCodeResponse{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return DeviceCodeResponse{}, fmt.Errorf("codex device code request failed: %s: %s", resp.Status, string(respBody))
	}

	var userCode codexUserCodeResponse
	if err := json.Unmarshal(respBody, &userCode); err != nil {
		return DeviceCodeResponse{}, err
	}
	code := userCode.UserCode
	if code == "" {
		code = userCode.Usercode
	}
	if userCode.DeviceAuthID == "" || code == "" {
		return DeviceCodeResponse{}, fmt.Errorf("codex device code response was missing required fields")
	}

	return DeviceCodeResponse{
		DeviceCode:              userCode.DeviceAuthID,
		UserCode:                code,
		VerificationURI:         codexBaseURL() + "/codex/device",
		VerificationURIComplete: codexBaseURL() + "/codex/device",
		ExpiresIn:               int((15 * time.Minute).Seconds()),
		Interval:                codexPollingInterval(userCode.IntervalRaw),
	}, nil
}

func WaitForCodexAuth(device DeviceCodeResponse) (config.AuthVal, error) {
	interval := device.Interval
	if interval < 1 {
		interval = 5
	}
	deadline := time.Now().Add(15 * time.Minute)
	time.Sleep(time.Duration(interval) * time.Second)

	for time.Now().Before(deadline) {
		codeResp, done, err := pollCodexAuthToken(device)
		if err != nil {
			return config.AuthVal{}, err
		}
		if done {
			return exchangeCodexCodeForTokens(codeResp.AuthorizationCode, codeResp.CodeVerifier, codexBaseURL()+"/deviceauth/callback")
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
	return config.AuthVal{}, fmt.Errorf("codex device code expired")
}

func SaveCodexAuth(authVal config.AuthVal) error {
	if authVal.Type == "" {
		authVal.Type = "oauth"
	}
	if len(authVal.Models) == 0 {
		authVal.Models = config.DefaultCodexModels()
	}
	if err := config.UpdateAuthVal(config.CodexAuthProvider, authVal); err != nil {
		return err
	}
	return config.ReconcileAuthProviderModels(config.CodexAuthProvider, authVal.Models)
}

func pollCodexAuthToken(device DeviceCodeResponse) (codexTokenPollResponse, bool, error) {
	body, err := json.Marshal(map[string]string{
		"device_auth_id": device.DeviceCode,
		"user_code":      device.UserCode,
	})
	if err != nil {
		return codexTokenPollResponse{}, false, err
	}
	req, err := http.NewRequest("POST", codexAccountsURL()+"/deviceauth/token", bytes.NewReader(body))
	if err != nil {
		return codexTokenPollResponse{}, false, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "lightcode")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return codexTokenPollResponse{}, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return codexTokenPollResponse{}, false, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return codexTokenPollResponse{}, false, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return codexTokenPollResponse{}, false, fmt.Errorf("codex device auth failed: %s: %s", resp.Status, string(respBody))
	}

	var codeResp codexTokenPollResponse
	if err := json.Unmarshal(respBody, &codeResp); err != nil {
		return codexTokenPollResponse{}, false, err
	}
	if codeResp.AuthorizationCode == "" || codeResp.CodeVerifier == "" {
		return codexTokenPollResponse{}, false, errors.New("codex device auth response was missing authorization code fields")
	}
	return codeResp, true, nil
}

func exchangeCodexCodeForTokens(code string, codeVerifier string, redirectURI string) (config.AuthVal, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", codexOAuthClientID)
	form.Set("code_verifier", codeVerifier)

	req, err := http.NewRequest("POST", codexBaseURL()+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return config.AuthVal{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "lightcode")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return config.AuthVal{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return config.AuthVal{}, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return config.AuthVal{}, fmt.Errorf("codex token exchange failed: %s: %s", resp.Status, string(respBody))
	}

	var tokenResp struct {
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return config.AuthVal{}, err
	}
	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		return config.AuthVal{}, fmt.Errorf("codex token exchange did not return access and refresh tokens")
	}

	expires := time.Now().Unix() + tokenResp.ExpiresIn
	if tokenResp.ExpiresIn <= 0 {
		expires = jwtExpiresAtLocal(tokenResp.AccessToken)
	} else if tokenResp.ExpiresIn > int64((5 * time.Minute).Seconds()) {
		expires -= int64((5 * time.Minute).Seconds())
	}

	return config.AuthVal{
		Type:      "oauth",
		Access:    tokenResp.AccessToken,
		Refresh:   tokenResp.RefreshToken,
		Expires:   expires,
		AccountId: codexAccountID(tokenResp.IDToken, tokenResp.AccessToken),
		Models:    config.DefaultCodexModels(),
	}, nil
}

func codexBaseURL() string {
	return strings.TrimRight(codexAuthBaseURL, "/")
}

func codexAccountsURL() string {
	return codexBaseURL() + "/api/accounts"
}

func codexPollingInterval(raw any) int {
	switch v := raw.(type) {
	case float64:
		return int(v)
	case string:
		interval, _ := strconv.Atoi(strings.TrimSpace(v))
		return interval
	default:
		return 0
	}
}

type codexPKCE struct {
	codeVerifier string
}

func generateCodexPKCE() (codexPKCE, error) {
	verifier, err := randomBase64URL(64)
	if err != nil {
		return codexPKCE{}, err
	}
	return codexPKCE{codeVerifier: verifier}, nil
}

func codexCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomBase64URL(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func listenCodexCallback() (net.Listener, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", codexLoginDefaultPort))
	if err == nil {
		return listener, nil
	}
	fallback, fallbackErr := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", codexLoginFallbackPort))
	if fallbackErr == nil {
		return fallback, nil
	}
	return nil, fmt.Errorf("default callback port %d failed: %v; fallback callback port %d failed: %w", codexLoginDefaultPort, err, codexLoginFallbackPort, fallbackErr)
}

func codexAccountID(tokens ...string) string {
	for _, token := range tokens {
		claims := jwtOpenAIAuthClaims(token)
		if accountID, ok := claims["chatgpt_account_id"].(string); ok {
			return accountID
		}
	}
	return ""
}

func jwtExpiresAtLocal(token string) int64 {
	claims := jwtClaims(token)
	exp, ok := claims["exp"].(float64)
	if !ok || exp <= 0 {
		return 0
	}
	return int64(exp) - int64((5 * time.Minute).Seconds())
}

func jwtOpenAIAuthClaims(token string) map[string]any {
	claims := jwtClaims(token)
	nested, ok := claims["https://api.openai.com/auth"].(map[string]any)
	if ok {
		return nested
	}
	return claims
}

func jwtClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims
}
