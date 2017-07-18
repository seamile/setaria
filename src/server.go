package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

/*
directory tree
--------------
themes/
|
└── default/
    |
    ├── static/
    |   |
    │   ├── css/
    │   ├── img/
    │   └── js/
    |
    └── template/
*/
const (
	themeDir    = "themes"
	staticDir   = "static"
	templateDir = "template"
	staticURL   = "/static/"
)

type Server struct {
	home  string
	theme string

	templatePath string
	staticPath   string

	docs     []*Note
	docIndex map[string]*Note
	tagIndex map[string][]*Note

	templates map[string]*template.Template
}

func (s *Server) init(home, theme string) {
	EnsureDirs(home)
	s.home = home
	s.theme = theme

	// set the static and the template path
	themePath := filepath.Join(RunningDir(), themeDir, s.theme)
	if IsNotExist(themePath) {
		panic("not found the theme")
	} else {
		s.templatePath = filepath.Join(themePath, templateDir)
		s.staticPath = filepath.Join(themePath, staticDir)
	}

	// load docs
	s.docIndex = make(map[string]*Note)
	s.tagIndex = make(map[string][]*Note)
	s.loadDocs()

	s.loadTemplates()
}

func (s *Server) loadDocs() error {
	walk := func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && filepath.Ext(path) == ".note" {
			// parse *.note files
			note := new(Note)
			if err = note.ParseFile(path); err != nil {
				return err
			}
			// index the note pointer for Server
			s.docs = append(s.docs, note)
			s.docIndex[note.Slug] = note
			for _, tag := range note.Tags {
				s.tagIndex[tag] = append(s.tagIndex[tag], note)
			}
		}
		return nil
	}

	return filepath.Walk(s.home, walk)
}

func (s *Server) loadTemplates() error {
	base := filepath.Join(s.templatePath, "base.html")

	walk := func(path string, info os.FileInfo, err error) error {
		name := info.Name()
		if filepath.Ext(name) == "html" && name != "base.html" {
			tmpl, err := template.New(name).Funcs(Funcs).ParseFiles(base, path)
			if err != nil {
				return err
			}
			s.templates[name] = tmpl
		}
		return nil
	}

	return filepath.Walk(s.templatePath, walk)
}

func (s *Server) run(host string, port int) error {
	// URL mapping

	// handle static files
	fileServer := http.FileServer(http.Dir(s.staticPath))
	fileServer = http.StripPrefix(staticURL, fileServer)
	http.Handle(staticURL, fileServer)

	// run server
	addr := fmt.Sprintf("%s:%d", host, port)
	return http.ListenAndServe(addr, nil)
}
