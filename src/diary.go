package main

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	rDate = regexp.MustCompile(`\d{4}-(0[1-9]|1[0-2])-[0-3]\d`)
	// Top Level
	rHeader  = regexp.MustCompile(`^# +(.+\S)`)           // # This is Header
	rWeather = regexp.MustCompile(`^Weather: +(.+\S)`)    // Weather: 🌞
	rTag     = regexp.MustCompile(`^Tags: +(.+\S)`)       // Tags: tag1 tag2 tag3
	rBlank   = regexp.MustCompile(`(?:^$|^ +$|^[\n\r]$)`) // BlankLine
	rHr      = regexp.MustCompile(`-{3,}`)                // ---
	// Blocks
	rOl        = regexp.MustCompile(`(?:^|^[ \t]+)\d+\. +(?P<AA>.+\S)`) // 0. OrderList
	rUl        = regexp.MustCompile(`(?:^|^[ \t]+)[*+-] +(.+\S)`)       // * UnorderList
	rQuote     = regexp.MustCompile(`(?:^|^[ \t]+)> +(.+\S)`)           // > QuoteBlock
	rCodeBlock = regexp.MustCompile("(?:^|^[ \t]+)``` +(\\w+)$")        // ```Programe
	rCodeEnd   = regexp.MustCompile("(?:^|^[ \t]+)```$")                // ```
	// Inline
	rStrong = regexp.MustCompile(`\*\*(.+)\*\*`)      // **strong**
	rCode   = regexp.MustCompile("`(.+)`")            // *code*
	rLink   = regexp.MustCompile(`\[(.+)\]\((.+)\)`)  // [img](http://example.com/)
	rImg    = regexp.MustCompile(`!\[(.+)\]\((.+)\)`) // ![img](http://example.com/img.jpg)
	rWord   = regexp.MustCompile(`\S+`)
)

type Diary struct {
	Title   string
	Date    string
	Content []string
	Weather string
	Tags    []string
	Slug    string // 链接名

	Tmpl template.HTML
}

// 解析文件
func (d *Diary) ParseFile(path string) error {
	// 解析文件名
	if err := d.parsePath(path); err != nil {
		return err
	}

	// 读取文件
	if bytes, err := ioutil.ReadFile(path); err != nil {
		return err
	} else {
		d.Content = strings.Split(string(bytes), "\n")
	}

	if err := d.parseContent(d.Content); err != nil {
		return err
	}

	return nil
}

// 通过文件名解析日期、Slug
// 文件名格式: 2016-06-03_HelloWord.md
func (d *Diary) parsePath(path string) error {
	_, filename := filepath.Split(path)

	// 解析日期
	d.Date = rDate.FindString(filename)
	if d.Date == "" {
		return errors.New(fmt.Sprintf("Wrong filename format: %s", filename))
	}

	// 解析 Slug
	d.Slug = slugify(filename)
	return nil
}

// 将文件名 slug 化
func slugify(filename string) string {
	slug := strings.TrimSuffix(filename, filepath.Ext(filename))
	r := regexp.MustCompile(`\W+`)
	return r.ReplaceAllString(slug, "")
}

type BlockType int

const (
	Body BlockType = iota
	OrderList
	UnorderList
	Quote
	CodeBlock
)

// 上下文状态
type Block struct {
	bType    BlockType
	nLevel   int    // Block 层级
	nBlank   int    // 连续空行数
	preBlock *Block // 前一个 Block
}

func (block *Block) close() {}

func (d *Diary) parseContent(lines []string) error {
	block := Block{bType: Body}

	for num, line := range lines {
		switch {
		case num == 0:
			d.Title = strings.TrimSpace(line)

		case rHeader.MatchString(line):
			// TODO: set header HTML
			rHeader.FindStringSubmatch(line)

		case rWeather.MatchString(line):
			d.Weather = rWeather.FindStringSubmatch(line)[1]

		case rTag.MatchString(line):
			d.Tags = rWord.FindAllString(rTag.FindStringSubmatch(line)[1], -1)

		case rHr.MatchString(line):
			// TODO: <hr>

		case rBlank.MatchString(line):
			block.nBlank++
			if block.nBlank >= 2 {
				// TODO: close current Block and open a new one
			}

		case rOl.MatchString(line):
			rOl.FindStringSubmatch(line)

		case rUl.MatchString(line):
			rUl.FindStringSubmatch(line)

		case rQuote.MatchString(line):
			rQuote.FindStringSubmatch(line)

		case rCode.MatchString(line):
			rCode.FindStringSubmatch(line)

		default:
			// <br>
		}
	}
	return nil
}

func (d *Diary) parseStrong(text string) {}

// 列表
func (d *Diary) parseList() {}

// 代码
func (d *Diary) parseCode() {}

// 段落
func (d *Diary) parseSection() {}

// 链接
func (d *Diary) parseLink() {}

// 图像
func (d *Diary) parseImage() {}

////////////////////////////////////////////////////////////////////////////////
func _test() {
	// rTag.FindAllStringSubmatch(s, n)
	for i, w := range rTag.FindStringSubmatch("Tags:   多 云 转 晴 / 🌞 ") {
		println(i, w)
	}
}

func main() {
	_test()
}
