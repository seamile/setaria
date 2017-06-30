package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ElemName string

const (
	// Element names
	root        ElemName = "Note"
	header      ElemName = "Header"
	strong      ElemName = "Strong"
	link        ElemName = "Link"
	img         ElemName = "Img"
	code        ElemName = "Code"
	precode     ElemName = "PreCode"
	orderlist   ElemName = "Ol"
	unorderlist ElemName = "Ul"
	blockquote  ElemName = "BlockQuote"
	paragraph   ElemName = "P"

	indent = 4 // indent spaces in note files
)

var (
	rDate  = regexp.MustCompile(`\d{4}-(0[1-9]|1[0-2])-([0-2]\d)|3[01]`)
	rBlank = regexp.MustCompile(`^[ \t]{0,}$`) // BlankLine

	rWeather = regexp.MustCompile(`^Weather: {1,3}(\S[\S ]{0,})`) // Weather: 🌞
	rTags    = regexp.MustCompile(`^Tags: {1,3}(\S[\S ]{0,})`)    // Tags: name1 name2 name3
	rWord    = regexp.MustCompile(`\S+`)

	rHeader = regexp.MustCompile(`^#{1,6} {1,3}(\S[\S ]{0,})`) // # This is Header
	rHr     = regexp.MustCompile(`^[-_*]{3,}`)                 // --- (wrapt by blank lines)

	rOl         = regexp.MustCompile(`^[ \t]{0,}\d+\. {1,3}(\S[\S ]{0,})`)           // 1. OrderList
	rUl         = regexp.MustCompile(`^[ \t]{0,}[*+-] {1,3}(\S[\S ]{0,})`)           // * UnorderList
	rBulletList = regexp.MustCompile(`^[ \t]{0,}(?:[*+-]|\d+\.) {1,3}(\S[\S ]{0,})`) // * UnorderList

	rBlockQuote = regexp.MustCompile(`(?:^[ \t]{0,})> {1,3}(\S[\S ]{0,})`) // > BlockQuote

	rPreCodeHead = regexp.MustCompile("(?:^[ \t]{0,})``` {0,3}(\\w{0,})") // ```Golang
	rPreCodeTail = regexp.MustCompile("(?:^[ \t]{0,})```$")               // ```

	rCode   = regexp.MustCompile("``(.+)``|`(.+)`") // *code*
	rStrong = regexp.MustCompile(`\*\*(.+)\*\*`)    // **strong**

	rLink = regexp.MustCompile(`\[([\S ]+)\]\(([\S ]+)\)`)  // [img](http://example.com/)
	rImg  = regexp.MustCompile(`!\[([\S ]+)\]\(([\S ]+)\)`) // ![img](http://example.com/img.jpg)

	elementTemplate *template.Template
)

func init() {
	var err error
	elementTemplate, err = template.ParseFiles("./template/elements.tmpl")
	if err != nil {
		fmt.Printf("Setaria: %s\n", err)
		os.Exit(1)
	}
}

func renderHTML(name ElemName, data interface{}) (string, error) {
	b := new(bytes.Buffer)
	err := elementTemplate.ExecuteTemplate(b, string(name), data)

	if err != nil {
		return "", err
	} else {
		return b.String(), nil
	}
}

type Lines struct {
	num     int
	blanks  []int
	content []string
}

func (l *Lines) length() int {
	return len(l.content)
}

func (l *Lines) current() string {
	return l.content[l.num]
}

func (l *Lines) line(n int) string {
	return l.content[n]
}

func (l *Lines) jumpto(n int) string {
	if 0 <= n && n < l.length() {
		l.num += n
	} else {
		l.num = 0
	}
	return l.current()
}

func (l *Lines) isBlank(num int) bool {
	for _, n := range l.blanks {
		if num == n {
			return true
		}
	}
	return false
}

func (l *Lines) recordBlank(num int) {
	if !l.isBlank(num) {
		l.blanks = append(l.blanks, num)
	}
}

type Block struct {
	name   ElemName
	level  int
	blanks int

	items  []string
	buffer []string

	html string
	prev *Block
}

func (b *Block) spawn(name ElemName) *Block {
	return &Block{name: name, level: b.level + 1, prev: b}
}

func (b *Block) addToBuffer(str string) {
	b.buffer = append(b.buffer, str)
}

func (b *Block) resetBuffer() {
	if len(b.buffer) > 0 {
		item := strings.Join(b.buffer, "\n")
		b.items = append(b.items, item)
		b.buffer = b.buffer[:0] // reset the buf
	}
}

func (b *Block) render() error {
	b.resetBuffer()
	if len(b.items) > 0 {
		html, err := renderHTML(b.name, b.items)
		if err != nil {
			return err
		} else {
			b.html = html
		}
	}
	return nil
}

func getLevel(spaces string, indent int) int {
	level, acc := 0, 0
	for _, chr := range spaces {
		switch {
		case chr == ' ':
			acc++
		case chr == '\t':
			acc = indent
		default:
			break
		}
		if acc == indent {
			acc = 0
			level++
		}
	}
	if 0 < acc && acc < indent {
		level++
	}
	return level
}

type Note struct {
	Title   string
	Date    string
	Auth    string
	Weather string
	Tags    []string
	Body    template.HTML

	Slug    string
	Summary string
}

// Parse a *.note file into note.HTML
func (note *Note) ParseFile(path string) error {
	// parse the filename
	if err := note.parsePath(path); err != nil {
		return err
	}
	// read the file
	lines := new(Lines)
	if bytes, err := ioutil.ReadFile(path); err != nil {
		return err
	} else {
		lines.content = strings.Split(string(bytes), "\n")
	}
	// parse
	if err := note.parseContent(lines); err != nil {
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

func (note *Note) parseContent(lines *Lines) error {
	block := &Block{name: root, level: 0}

	for cnt, line := lines.length(), ""; lines.num < cnt; lines.num++ {
		print("Line: ", lines.num, " -> ")
		switch line = lines.current(); {
		case lines.num == 0:
			note.Title = strings.TrimSpace(line)
			print("Title  : ", note.Title)
		case rWeather.MatchString(line):
			note.Weather = rWeather.FindStringSubmatch(line)[1]
			print("Weather: ", note.Weather)
		case rTags.MatchString(line):
			note.Tags = rWord.FindAllString(rTags.FindStringSubmatch(line)[1], -1)
			print("Tags   : ", note.Tags)

		case rHeader.MatchString(line):
			print("Header : ", rHeader.FindStringSubmatch(line)[1])
			html, err := renderHTML(header, rHeader.FindStringSubmatch(line)[1])
			if err != nil {
				return err
			}
			block.addToBuffer(html)

		case rHr.MatchString(line):
			block.addToBuffer("\n<hr>\n")
			print("Hr    : ", line)

		case rPreCodeHead.MatchString(line):
			print("PreCode: ", line)
			son := block.spawn(precode)
			if err := parsePreCode(son, lines); err != nil {
				return err
			}
			block.addToBuffer(son.html)

		case rBlockQuote.MatchString(line):
			print("Quote  : ", line)
			son := block.spawn(blockquote)
			if err := parseBlockQuote(son, lines); err != nil {
				return err
			}
			block.addToBuffer(son.html)
			print(block.html)

		case rOl.MatchString(line):
			print("Ol     : ", line)
			son := block.spawn(orderlist)
			if err := parseBulletList(son, lines); err != nil {
				return err
			}
			block.addToBuffer(son.html)

		case rUl.MatchString(line):
			print("Ul     : ", line, "  |  ", lines.num)
			son := block.spawn(unorderlist)
			if err := parseBulletList(son, lines); err != nil {
				return err
			}
			block.addToBuffer(son.html)
			print("  |  ", lines.num)
		}
		// println(line)
		println()
	}
	// println("\n\nHTML:\n", block.html, "\nEND\n")
	note.Body = template.HTML(block.html)
	return nil
}

func parsePreCode(block *Block, lines *Lines) error {
	lang := ""
	codes := []string{}

	// check
	if line := lines.current(); !rPreCodeHead.MatchString(line) {
		return errors.New("Not match PreCode")
	} else {
		// get the lang
		lang = rPreCodeHead.FindStringSubmatch(line)[1]
		lines.num++
		// pick out the codes
		for cnt := lines.length(); lines.num < cnt; lines.num++ {
			line = lines.current()
			if rPreCodeTail.MatchString(line) {
				break
			} else {
				codes = append(codes, line)
			}
		}
	}

	data := struct {
		Codes string
		Lang  string
	}{strings.Join(codes, "\n"), lang}

	html, err := renderHTML(precode, data)
	block.html = html
	return err
}

func parseBlockQuote(block *Block, lines *Lines) error {
loop:
	for cnt, line := lines.length(), ""; lines.num < cnt; lines.num++ {
		switch line = lines.current(); {
		case rBlockQuote.MatchString(line):
			block.addToBuffer(rBlockQuote.FindStringSubmatch(line)[1])
		case rBlank.MatchString(line):
			block.resetBuffer()
		default:
			break loop
		}

	}

	return block.render()
}

func parseBulletList(block *Block, lines *Lines) error {
	var line string
loop:
	for cnt := lines.length(); lines.num < cnt; lines.num++ {
		switch line = lines.current(); {
		case rBulletList.MatchString(line):
			block.blanks = 0
			lv := getLevel(line, indent)
			switch {
			case block.level == lv:
				block.resetBuffer()
				block.addToBuffer(rBulletList.FindStringSubmatch(line)[1])
			case block.level < lv: // son block level
				son := block.spawn(block.name)
				parseBulletList(son, lines)
				block.addToBuffer(son.html)
			case block.level > lv: // father block level
				break loop
			}

		case rBlank.MatchString(line):
			block.resetBuffer()
			block.blanks++
		default:
			if block.blanks == 0 {
				block.addToBuffer(line)
			} else {
				block.render()
				break loop
			}
			// reset blanks
			block.blanks = 0
		}
	}
	block.render()
	return nil
}

// test code
func _test() {
	note := new(Note)
	if err := note.ParseFile("../_test/2017-04-09_演示文稿.note"); err != nil {
		println("Err:", err)
	} else {
		println("Body:", note.Body)
	}
}

func main() {
	// rWeather = regexp.MustCompile(`Weather: {1,3}(\S[\S ]{0,})`) // Weather: 🌞
	// s := "Weather: 晴"
	// println(rWeather.MatchString(s))
	_test()
}
