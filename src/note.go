package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	rDate = regexp.MustCompile(`\d{4}-(0[1-9]|1[0-2])-([0-2]\d)|3[01]`)

	rHeader      = regexp.MustCompile(`^#{1,6} {1,3}(\S[\S ]+)`)             // # This is Header
	rHr          = regexp.MustCompile(`^[-_*]{3,}`)                          // --- (wrapt by blank lines)
	rWeather     = regexp.MustCompile(`^Weather: {1,3}(\S[\S ]+)`)           // Weather: 🌞
	rTag         = regexp.MustCompile(`^Tags: {1,3}(\S[\S ]+)`)              // Tags: name1 name2 name3
	rBlank       = regexp.MustCompile(`^[ \t]{0,}$`)                         // BlankLine
	rOl          = regexp.MustCompile(`(?:^[ \t]{0,})\d+\. {1,3}(\S[ \S]+)`) // 1. OrderList
	rUl          = regexp.MustCompile(`(?:^[ \t]{0,})[*+-] {1,3}(\S[ \S]+)`) // * UnorderList
	rQuoteBlock  = regexp.MustCompile(`(?:^[ \t]{0,})> {1,3}(\S[ \S]+)`)     // > QuoteBlock
	rPreCodeHead = regexp.MustCompile("(?:^[ \t]{0,})``` {1,3}(\\w+)$")      // ```Golang
	rPreCodeTail = regexp.MustCompile("(?:^[ \t]{0,})```$")                  // ```
	rStrong      = regexp.MustCompile(`\*\*(.+)\*\*`)                        // **strong**
	rCode        = regexp.MustCompile("`(.+)`")                              // *code*
	rLink        = regexp.MustCompile(`\[([\S ]+)\]\(([\S ]+)\)`)            // [img](http://example.com/)
	rImg         = regexp.MustCompile(`!\[([\S ]+)\]\(([\S ]+)\)`)           // ![img](http://example.com/img.jpg)
	rWord        = regexp.MustCompile(`\S+`)
)

type Note struct {
	Title   string
	Date    string
	Auth    string
	Weather string
	Tags    []string
	Body    template.HTML

	Slug    string
	Summary string
	content []string
}

// Parse a *.note file into note.HTML
func (note *Note) ParseFile(path string) error {
	// parse the filename
	if err := note.parsePath(path); err != nil {
		return err
	}
	// read the file
	if bytes, err := ioutil.ReadFile(path); err != nil {
		return err
	} else {
		note.content = strings.Split(string(bytes), "\n")
	}
	// parse
	if err := note.parseContent("TODO: tmpl_path"); err != nil {
		return err
	}
	return nil
}

// Get date and slug from the filename
// Filename format: 2016-06-03_HelloWord.md
func (note *Note) parsePath(path string) error {
	_, filename := filepath.Split(path)
	// parse date
	note.Date = rDate.FindString(filename)
	if note.Date == "" {
		return errors.New(fmt.Sprintf("Wrong filename format: %s", filename))
	}
	// parse slug
	note.Slug = slugify(filename)
	return nil
}

func slugify(filename string) string {
	slug := strings.TrimSuffix(filename, filepath.Ext(filename))
	r := regexp.MustCompile(`\W+`)
	return r.ReplaceAllString(slug, "_")
}

type Elem int

const (
	Root        Elem = iota
	PlainText        = iota
	BlankLine        = iota
	OrderList        = iota
	UnorderList      = iota
	QuoteBlock       = iota
	PreCode          = iota
)

type Block struct {
	etype  Elem
	level  int
	blanks int
	prev   *Block
}

type BulletList struct {
	name   string
	bullet []string
}

func (note *Note) parseContent(filenames ...string) error {
	block := Block{etype: Root, level: 0, blanks: 0}
	tmpl, err := template.ParseFiles(filenames...)

	for num, line := range note.content {
		switch {
		case num == 0:
			note.Title = strings.TrimSpace(line)

		case rHeader.MatchString(line):
			header := rHeader.FindStringSubmatch(line)[1]
			// renderHTML(t, name, data)

		case rWeather.MatchString(line):
			note.Weather = rWeather.FindStringSubmatch(line)[1]

		case rTag.MatchString(line):
			note.Tags = rWord.FindAllString(rTag.FindStringSubmatch(line)[1], -1)

		case rHr.MatchString(line):
			// TODO: <hr>

		case rBlank.MatchString(line):
			block.blanks++
			if block.blanks >= 2 {
				// TODO: close current Block and open a new one
			}

		case rOl.MatchString(line):
			rOl.FindStringSubmatch(line)

		case rUl.MatchString(line):
			rUl.FindStringSubmatch(line)

		case rQuoteBlock.MatchString(line):
			rQuoteBlock.FindStringSubmatch(line)

		case rCode.MatchString(line):
			rCode.FindStringSubmatch(line)

		default:
			// <br>
		}
	}
	return nil
}

func (note *Note) parseStrong(text string) {}

// 列表
func (note *Note) parseList() {}

// 代码
func (note *Note) parseCode() {}

// 段落
func (note *Note) parseSection() {}

// 链接
func (note *Note) parseLink() {}

// 图像
func (note *Note) parseImage() {}

func renderHTML(t *template.Template, name string, data interface{}) (template.HTML, error) {
	b := new(bytes.Buffer)
	err := t.ExecuteTemplate(b, name, data)

	if err != nil {
		return "", err
	} else {
		return template.HTML(b.String()), nil
	}
}

///////////////////////////////// test code ////////////////////////////////////

func _test() {
	for i, s := range rHeader.FindStringSubmatch("# 大王叫我来巡山 - 阿斯蒂芬") {
		println(i, "->", s)
	}
	// rTag.FindAllStringSubmatch(s, n)
	// for i, w := range rTag.FindStringSubmatch("Tags:   多 云 转 晴 / 🌞 ") {
	// 	println(i, w)
	// }

}

func main() {
	_test()
}
