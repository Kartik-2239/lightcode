package main

import (
	"math/rand"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server"
)

// "github.com/Kartik-2239/lightcode/internal/server"

func main() {
	server.Initialise()
	// for i := 0; i < 2; i++ {
	// 	session_id := randomSessionID()
	// 	agent.New().Run("list my directory and tell me something interesting about one random file in it, the file should be small, don't read env at all", session_id)
	// }
}

func randomSessionID() string {
	var chars = "qwertyuiopasdfghjklzxcvbnmQWERTYUIOPASDFGHJKLZXCVBNM1234567890-_"
	length := 10
	var result strings.Builder
	for range length {
		result.WriteString(string(chars[rand.Intn(len(chars))]))
	}
	return result.String()
}
