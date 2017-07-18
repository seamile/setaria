package main

import (
	"net/http"
	// "path/filepath"
)

type IndexData struct {
	Title string
	Docs  []*Note
}

func Home(w http.ResponseWriter, r *http.Request) {
	// data := IndexData{"Blog", server.docs}
	// server.tmpl.home.ExecuteTemplate(w, "blog", data)
}

func Page(w http.ResponseWriter, r *http.Request) {
	// _, slug := filepath.Split(r.URL.Path)
	// article, ok := server.docIndex[slug]
	// if ok {
	// 	server.tmpl.page.ExecuteTemplate(w, "blog", article)
	// } else {
	// 	http.Redirect(w, r, "/", http.StatusFound)
	// }
}

// TODO: keyword search and tag filter
func Filter(w http.ResponseWriter, r *http.Request) {}
