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

// 配置项
type Config struct {
	HTTPAddr string // 地址
	BaseDir  string // 根目录 (绝对路径)

	ArticleDir  string // 文章目录 (根目录的相对路径)
	TemplateDir string // 模板目录 (根目录的相对路径)
	StaticDir   string // 静态文件目录 (根目录的相对路径)
	StaticURL   string // 静态文件URL
}

// 初始化配置
func (c *Config) Initialize(httpAddr string, baseDir string) {
	c.HTTPAddr = httpAddr
	if filepath.IsAbs(baseDir) {
		c.BaseDir = baseDir
	} else {
		c.BaseDir, _ = filepath.Abs(baseDir)
	}

	c.ArticleDir = filepath.Join(c.BaseDir, "data/articles")
	c.TemplateDir = filepath.Join(c.BaseDir, "templates")
	c.StaticDir = filepath.Join(c.BaseDir, "statics")
	c.StaticURL = "/static/"
}

// TODO: 从 JSON 配置文件初始化
func (c *Config) LoadFromJSON(JSONFile string) {}

// Blog Server
type Server struct {
	config  Config
	docs    []*Article
	docMap  map[string]*Article
	tagDocs map[string][]*Article
	tmpl    struct {
		home   *template.Template
		filter *template.Template
		page   *template.Template
	}
}

// 初始化 Server
func (s *Server) Initialize(httpAddr string, baseDir string) error {
	s.config.Initialize(httpAddr, baseDir)
	s.docMap = make(map[string]*Article)
	s.tagDocs = make(map[string][]*Article)
	// 遍历文章目录，加载文章数据
	if err := filepath.Walk(s.config.ArticleDir, s.parseArticle); err != nil {
		panic(err)
	}
	// 初始化模板
	s.parseTemplates()

	// TODO: 添加身份验证 handler
	// 身份验证只需 POST 一段有一定复杂度的密码即可
	// 密码由管理员手动生成，加密保存，既是身份识别又用做安全验证
	// http.HandleFunc(s.config.AuthURL, auth)

	return nil
}

// 解析文章
func (s *Server) parseArticle(path string, info os.FileInfo, err error) error {
	if !info.IsDir() && filepath.Ext(path) == ".txt" {
		a := Article{}
		if err = a.Load(path); err != nil {
			return err
		}
		s.docs = append(s.docs, &a)
		s.docMap[a.Link] = &a
		for _, tag := range a.Tags {
			s.tagDocs[tag] = append(s.tagDocs[tag], &a)
		}
	}
	return nil
}

// 解析模板
func (s *Server) parseTemplates() {
	base := filepath.Join(s.config.TemplateDir, "base.html")
	parse := func(name string) (*template.Template, error) {
		t := template.New("").Funcs(template.FuncMap{"HTML": articleToHTML})
		return t.ParseFiles(base, filepath.Join(s.config.TemplateDir, name))
	}

	var err error
	if s.tmpl.home, err = parse("home.html"); err != nil {
		panic(err)
	}
	if s.tmpl.page, err = parse("page.html"); err != nil {
		panic(err)
	}
	if s.tmpl.filter, err = parse("filter.html"); err != nil {
		panic(err)
	}
}

// 检查文件名是否含有特殊字符
func validFilename(filename string) bool {
	return !strings.ContainsAny(filename, `"'*/:<>?\\|`)
}

// TODO: 保存文章
func (s *Server) Save(a *Article) {
	if a.Title != "" && a.Date != "" {
		titleField := strings.Replace(a.Title, " ", "_", -1)
		dateField := a.Date // TODO: 格式化为 yyyymmdd
		filename := fmt.Sprintf("%s_%s.txt", dateField, titleField)
		if validFilename(filename) {
			path := filepath.Join(s.config.ArticleDir, filename)
			print(path)
		}
	}
}

// 将文章内容转换为 HTML
func articleToHTML(text string) template.HTML {
	text = strings.Replace(text, "\n", "<br>", -1)
	return template.HTML(text)
}

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
