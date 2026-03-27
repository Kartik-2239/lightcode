package main

import (
	"net"
	"net/http"
	"os"

	"github.com/Kartik-2239/lightcode/internal/server"
	"github.com/Kartik-2239/lightcode/internal/server/config"
	"github.com/Kartik-2239/lightcode/internal/tui/views"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load(config.EnvPath())
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if !isPortInUse(port) {
		_, err := http.Get("http://localhost:" + port)
		if err != nil {
			ready := make(chan struct{})
			go server.Initialise(ready, port)
			<-ready
		}
		// body, err := io.ReadAll(resp.Body)
		// if string(body) != "lightcode is running!" {
		// 	log.Fatal("port: " + port + " is not being used by lightcode!")
		// }
	}
	views.LauchHomePage()
}

func isPortInUse(port string) bool {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return true // port is in use
	}
	ln.Close()
	return false
}
