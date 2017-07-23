package main

import (
	"flag"
	"log"
)

var (
	host  = flag.String("host", "localhost", "your host ip or hostname")
	port  = flag.Int("port", 8000, "the server port for listening")
	notes = flag.String("notes", "./SetariaNotes", "the blog files' storage path")
	theme = flag.String("theme", "simple", "blog theme")

	server = new(Server)
)

func main() {
	flag.Parse()
	server.Init(*notes, *theme)
	log.Fatal(server.Run(*host, *port))
}
