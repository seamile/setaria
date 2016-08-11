package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
)

type IndexData struct {
	Title string
	Docs  []*Article
}

var (
	server   Server
	httpAddr = flag.String("http", "localhost:8000", "HTTP 监听地址")
	baseDir  = flag.String("base", "./", "项目根路径")
)

// 首页
func home(w http.ResponseWriter, r *http.Request) {
	data := IndexData{"Blog", server.docs}
	server.tmpl.home.ExecuteTemplate(w, "blog", data)
}

// 文章页
func page(w http.ResponseWriter, r *http.Request) {
	_, linkname := filepath.Split(r.URL.Path)
	article, ok := server.docMap[linkname]
	if ok {
		server.tmpl.page.ExecuteTemplate(w, "blog", article)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// TODO: 文章过滤列表: search / tag 等
func filter(w http.ResponseWriter, r *http.Request) {}

func main() {
	// 初始化配置
	flag.Parse()
	server.Initialize(*httpAddr, *baseDir)

	// URLs
	http.HandleFunc("/", home)
	http.HandleFunc("/page/", page)
	http.HandleFunc("/filter", filter)

	// 静态文件处理
	fileServer := http.FileServer(http.Dir(server.config.StaticDir))
	fileServer = http.StripPrefix(server.config.StaticURL, fileServer)
	http.Handle(server.config.StaticURL, fileServer)

	// 启动服务器
	println("Server Start")
	log.Fatal(http.ListenAndServe(server.config.HTTPAddr, nil))
}
