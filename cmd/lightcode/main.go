package main

import (
	"github.com/Kartik-2239/lightcode/internal/server"
	"github.com/Kartik-2239/lightcode/internal/tui/views"
)

func main() {
	ready := make(chan struct{})
	go server.Initialise(ready)
	<-ready
	views.LauchHomePage()
}
