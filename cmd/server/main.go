package main

import "github.com/Kartik-2239/lightcode/internal/server"

func main() {
	port := "8080"
	ready := make(chan struct{})
	go server.Initialise(ready, port)
	<-ready
}
