package main

import "github.com/Kartik-2239/lightcode/internal/server"

func main() {
	ready := make(chan struct{})
	go server.Initialise(ready)
	<-ready
}
