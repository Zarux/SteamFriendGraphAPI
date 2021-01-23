package main

import (
	"flag"
	"sfgapi/internal/server"
)

var port = flag.String("port", "8080", "")

func main() {
	s := server.New(*port)
	s.Serve()
}
