package oauth

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestStartCodexAuthFlow(t *testing.T) {
	originalBaseURL := codexAuthBaseURL
	defer func() { codexAuthBaseURL = originalBaseURL }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/accounts/deviceauth/usercode" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var req map[string]string
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req["client_id"] != codexOAuthClientID {
			t.Fatalf("unexpected client id %q", req["client_id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_auth_id":"device-id","usercode":"ABCD-EFGH","interval":"2"}`))
	}))
	defer server.Close()
	codexAuthBaseURL = server.URL

	device, err := StartCodexAuthFlow()
	if err != nil {
		t.Fatalf("StartCodexAuthFlow returned error: %v", err)
	}
	if device.DeviceCode != "device-id" || device.UserCode != "ABCD-EFGH" {
		t.Fatalf("unexpected device response %#v", device)
	}
	if device.VerificationURI != server.URL+"/codex/device" {
		t.Fatalf("unexpected verification URI %q", device.VerificationURI)
	}
	if device.Interval != 2 {
		t.Fatalf("expected interval 2, got %d", device.Interval)
	}
}

func TestPollCodexAuthTokenAndExchange(t *testing.T) {
	originalBaseURL := codexAuthBaseURL
	defer func() { codexAuthBaseURL = originalBaseURL }()

	accessToken := codexTestJWT(t, map[string]any{"exp": time.Now().Add(time.Hour).Unix()})
	idToken := codexTestJWT(t, map[string]any{
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "account-id",
		},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/accounts/deviceauth/token":
			var req map[string]string
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode poll request: %v", err)
			}
			if req["device_auth_id"] != "device-id" || req["user_code"] != "ABCD-EFGH" {
				t.Fatalf("unexpected poll request %#v", req)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"authorization_code":"auth-code","code_challenge":"challenge","code_verifier":"verifier"}`))
		case "/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("grant_type") != "authorization_code" {
				t.Fatalf("unexpected grant type %q", r.Form.Get("grant_type"))
			}
			if r.Form.Get("code") != "auth-code" || r.Form.Get("code_verifier") != "verifier" {
				t.Fatalf("unexpected token form %s", r.Form.Encode())
			}
			if !strings.HasSuffix(r.Form.Get("redirect_uri"), "/deviceauth/callback") {
				t.Fatalf("unexpected redirect uri %q", r.Form.Get("redirect_uri"))
			}
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]any{
				"id_token":      idToken,
				"access_token":  accessToken,
				"refresh_token": "refresh-token",
				"expires_in":    3600,
			}
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				t.Fatalf("encode response: %v", err)
			}
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	codexAuthBaseURL = server.URL

	codeResp, done, err := pollCodexAuthToken(DeviceCodeResponse{DeviceCode: "device-id", UserCode: "ABCD-EFGH"})
	if err != nil {
		t.Fatalf("pollCodexAuthToken returned error: %v", err)
	}
	if !done {
		t.Fatal("expected polling to complete")
	}
	authVal, err := exchangeCodexCodeForTokens(codeResp.AuthorizationCode, codeResp.CodeVerifier, server.URL+"/deviceauth/callback")
	if err != nil {
		t.Fatalf("exchangeCodexCodeForTokens returned error: %v", err)
	}
	if authVal.Access != accessToken || authVal.Refresh != "refresh-token" {
		t.Fatalf("unexpected auth tokens %#v", authVal)
	}
	if authVal.AccountId != "account-id" {
		t.Fatalf("expected account id, got %q", authVal.AccountId)
	}
}

func TestCodexBrowserAuthURLIncludesPKCEAndLocalCallback(t *testing.T) {
	flow := &CodexBrowserAuthFlow{
		redirectURI:  "http://localhost:1455/auth/callback",
		codeVerifier: "test-verifier",
		state:        "state-value",
	}

	authURL, err := url.Parse(flow.AuthURL())
	if err != nil {
		t.Fatalf("auth URL did not parse: %v", err)
	}
	query := authURL.Query()
	if authURL.String() == "" || authURL.Path != "/oauth/authorize" {
		t.Fatalf("unexpected auth URL %s", authURL.String())
	}
	if query.Get("client_id") != codexOAuthClientID {
		t.Fatalf("unexpected client id %q", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != flow.redirectURI {
		t.Fatalf("unexpected redirect URI %q", query.Get("redirect_uri"))
	}
	if query.Get("code_challenge") != codexCodeChallenge(flow.codeVerifier) {
		t.Fatalf("unexpected code challenge %q", query.Get("code_challenge"))
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf("unexpected challenge method %q", query.Get("code_challenge_method"))
	}
	if query.Get("state") != flow.state {
		t.Fatalf("unexpected state %q", query.Get("state"))
	}
	if !strings.Contains(query.Get("scope"), "offline_access") {
		t.Fatalf("expected offline_access scope, got %q", query.Get("scope"))
	}
	if query.Get("codex_cli_simplified_flow") != "true" {
		t.Fatalf("expected simplified flow, got %q", query.Get("codex_cli_simplified_flow"))
	}
}

func codexTestJWT(t *testing.T, payload map[string]any) string {
	t.Helper()
	encode := func(v any) string {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("json marshal failed: %v", err)
		}
		return base64.RawURLEncoding.EncodeToString(data)
	}
	return encode(map[string]string{"alg": "none"}) + "." + encode(payload) + ".sig"
}
