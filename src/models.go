package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 文章
type Article struct {
	Title    string   // 标题
	Datetime string   // 时间
	Weather  string   // 天气
	Content  string   // 内容
	Summary  string   // 摘要
	Tags     []string // 标签
	Link     string
}

// 解析日期字符串
func (a Article) splitDateWeather(text string) (string, string) {
	re, _ := regexp.Compile(`\d{4}-[01]\d-[0-3]\d [01]\d:[0-5]\d`)
	idx := re.FindStringIndex(text)
	if len(idx) != 2 {
		return "", ""
	}
	datetime := text[idx[0]:idx[1]]             // 读取日期
	weather := strings.TrimSpace(text[idx[1]:]) // 天气
	return datetime, weather
}

// 解析正文和标签
func (a Article) splitContentTags(text string) (string, []string) {
	index := strings.LastIndex(text, "\n\nTags: ")
	if index == -1 {
		return text, []string{}
	}
	tags := strings.Split(text[index+8:], " ")
	return text[:index], tags
}

// 加载文章
func (a *Article) Load(articlePath string) error {
	bytes, err := ioutil.ReadFile(articlePath)
	if err != nil {
		return err
	}
	text := strings.SplitN(string(bytes), "\n\n", 3)
	a.Title = text[0]
	a.Datetime, a.Weather = a.splitDateWeather(text[1])
	a.Content, a.Tags = a.splitContentTags(text[2])
	a.setSummary(200, 5)
	_, filename := filepath.Split(articlePath)
	a.Link = strings.TrimRight(filename, ".txt")
	return nil
}

// 文章摘要
func (a *Article) setSummary(maxChar, maxLine int) {
	if content := []rune(a.Content); len(content) >= maxChar {
		a.Summary = string(content[:100])
	} else {
		a.Summary = a.Content
	}
	if i := 0; strings.Count(a.Summary, "\n") > maxLine {
		idx := strings.IndexFunc(a.Summary, func(c rune) bool {
			if c == '\n' {
				i++
			}
			return i >= maxLine
		})
		a.Summary = a.Summary[:idx]
	}
}

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
	if a.Title != "" && a.Datetime != "" {
		titleField := strings.Replace(a.Title, " ", "_", -1)
		dateField := a.Datetime // TODO: 格式化为 yyyymmdd
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
