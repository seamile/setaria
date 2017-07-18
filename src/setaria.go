package main

import (
	"flag"
	"log"
)

var (
	host  = flag.String("host", "localhost", "server listen address")
	port  = flag.Int("port", 8000, "server listen port")
	home  = flag.String("home", "./setaria", "home directory for notebook")
	theme = flag.String("theme", "default", "specify a theme for web style")

	server = new(Server)
)

func main() {
	flag.Parse()
	server.init(*home, *theme)
	log.Printf("Server running at %s:%d", *host, *port)
	log.Fatal(server.run(*host, *port))
}
