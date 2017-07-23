package main

import (
	"net/http"
	"path/filepath"
)

var URL_MAPPING = map[string]func(http.ResponseWriter, *http.Request){
	"/":        Home,
	"/home/":   Home,
	"/blog/":   Blog,
	"/filter/": Filter,
}

func Home(response http.ResponseWriter, request *http.Request) {
	data := struct {
		Title string
		Docs  []*Note
	}{"Blog", server.docs}
	server.Render(response, "home.html", data)
}

func Blog(response http.ResponseWriter, request *http.Request) {
	_, slug := filepath.Split(request.URL.Path)
	note, ok := server.docIndex[slug]
	if ok {
		server.Render(response, "blog.html", note)
	} else {
		http.Redirect(response, request, "/", http.StatusFound)
	}
}

// TODO: keyword search and tag filter
func Filter(response http.ResponseWriter, request *http.Request) {}
