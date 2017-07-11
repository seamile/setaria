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
	INDENT = 4 // indent spaces in note files

	// Element names
	EN_ROOT        ElemName = "Note"
	EN_TEXT        ElemName = "text"
	EN_HEADER      ElemName = "Header"
	EN_LINK        ElemName = "Link"
	EN_IMG         ElemName = "Img"
	EN_STRONG      ElemName = "Strong"
	EN_CODE        ElemName = "Code"
	EN_PRECODE     ElemName = "PreCode"
	EN_ORDERLIST   ElemName = "Ol"
	EN_UNORDERLIST ElemName = "Ul"
	EN_BLOCKQUOTE  ElemName = "BlockQuote"
	EN_PARAGRAPH   ElemName = "P"
)

var (
	ELEMENTS_TMPL *template.Template
	BR_FLAGS      = []byte(`.。"”!！?？…`) // symbols for line break
	HR            = []byte("\n<hr>\n")

	// Custom elements
	rWeather = regexp.MustCompile(`^Weather: {1,3}(\S[\S ]*)`) // Weather: 🌞 ⛅
	rAuth    = regexp.MustCompile(`^Auth: {1,3}(\S[\S ]*)`)    // Auth: Lee Bai
	rTags    = regexp.MustCompile(`^Tags: {1,3}(\S[\S ]*)`)    // Tags: name1 name2 name3
	// Standard elements
	rHeader      = regexp.MustCompile(`^#{1,6} {1,3}(\S[\S ]*)`)
	rHr          = regexp.MustCompile(`^(-{3,}|_{3,}|\*{3,})$`)
	rBulletList  = regexp.MustCompile(`^[ \t]*([*+-]|\d+\.) {1,3}(\S[\S ]*)`)
	rOl          = regexp.MustCompile(`^[ \t]*\d+\. {1,3}(\S[\S ]*)`)
	rUl          = regexp.MustCompile(`^[ \t]*[*+-] {1,3}(\S[\S ]*)`)
	rBlockQuote  = regexp.MustCompile(`^[ \t]*> {0,3}(\S[\S ]*|[ \t]*$)`)
	rPreCodeHead = regexp.MustCompile("(?:^[ \t]*)``` {0,3}(\\w*)")
	rPreCodeTail = regexp.MustCompile("(?:^[ \t]*)```$")
	rCode        = regexp.MustCompile("``(.+?)``|`(.+?)`")
	rStrong      = regexp.MustCompile(`\*\*(.+?)\*\*`)
	rLink        = regexp.MustCompile(`[^!]\[([\w ]*)\]\((\S*) *?\)`)
	rImg         = regexp.MustCompile(`!\[([\w ]*)\]\((\S*) *?\)`)
	// Others
	rDate  = regexp.MustCompile(`\d{4}-(0[1-9]|1[0-2])-([0-2]\d)|3[01]`)
	rWord  = regexp.MustCompile(`\S+`)
	rBlank = regexp.MustCompile(`^[ \t]*$`)
)

func init() {
	var err error
	ELEMENTS_TMPL, err = template.ParseFiles("./template/elements.tmpl")
	if err != nil {
		fmt.Printf("Setaria: %s\n", err)
		os.Exit(1)
	}
}

type Lines struct {
	num     int
	content []string
}

func (l *Lines) total() int {
	return len(l.content)
}

func (l *Lines) current() string {
	return l.content[l.num]
}

func (l *Lines) line(n int) string {
	return l.content[n]
}

func (l *Lines) jumpto(n int) string {
	line := l.content[n]
	l.num = n
	return line
}

type Block struct {
	name  ElemName
	level int

	children [][]template.HTML
	buffer   []template.HTML
	// buffer   *bytes.Buffer

	html template.HTML
	prev *Block
}

func (b *Block) spawn(name ElemName) *Block {
	return &Block{name: name, level: b.level + 1, prev: b}
}

// func (b *Block) buffer_check() {
// 	if b.buffer == nil {
// 		b.buffer = new(bytes.Buffer)
// 	}
// }

func (b *Block) buffer_write(content interface{}) (err error) {
	switch value := content.(type) {
	case string:
		// text = parseInLine(text)
		b.buffer = append(b.buffer, template.HTML(value))
	case template.HTML:
		b.buffer = append(b.buffer, value)
	}
	// b.buffer_check()
	// switch value := content.(type) {
	// case []byte:
	// 	_, err = b.buffer.Write(value)
	// case string:
	// 	_, err = b.buffer.WriteString(value)
	// case template.HTML:
	// 	_, err = b.buffer.WriteString(string(value))
	// default:
	// 	panic("Block.buffer_write: type error")
	// }
	return err
}

func (b *Block) buffer_dump() {
	// b.buffer_check()
	// if b.buffer.Len() > 0 {
	// 	b.children = append(b.children, b.buffer.String())
	// }
	// b.buffer.Reset()
}

func (b *Block) render() error {
	b.buffer_dump()
	if len(b.children) > 0 {
		html, err := renderHTML(b.name, b.children)
		if err != nil {
			return err
		} else {
			b.html = template.HTML(html)
		}
	}
	return nil
}

func (b *Block) backTo(level int) *Block {
	block := b
	for block.level > level && block.level >= 0 {
		if err := block.render(); err != nil {
			panic(err)
		}
		block.prev.buffer_write(block.html)
		block = block.prev
	}
	return block
}

func getLevel(line string, indent int) int {
	level, acc := 1, 0
LOOP:
	for _, chr := range line {
		switch {
		case chr == ' ':
			acc++
		case chr == '\t':
			acc = indent
		default:
			break LOOP
		}

		if acc == indent {
			acc = 0
			level++
		}
	}
	return level
}

func renderHTML(name ElemName, data interface{}) (string, error) {
	b := new(bytes.Buffer)
	err := ELEMENTS_TMPL.ExecuteTemplate(b, string(name), data)
	if err != nil {
		return "", err
	} else {
		return b.String(), nil
	}
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
	var line string
	block := &Block{name: EN_ROOT, level: 0}

	for cnt := lines.total(); lines.num < cnt; lines.num++ {
		line = lines.current()
		// println(lines.num, ":", line)
		switch {
		case lines.num == 0:
			note.Title = strings.TrimSpace(line)
			// println("	Title:", note.Title)

		case rWeather.MatchString(line):
			note.Weather = rWeather.FindStringSubmatch(line)[1]
			// println("	Weather:", note.Weather)

		case rAuth.MatchString(line):
			note.Auth = strings.TrimSpace(rAuth.FindStringSubmatch(line)[1])
			// println("	Auth:", note.Auth)

		case rTags.MatchString(line):
			note.Tags = rWord.FindAllString(rTags.FindStringSubmatch(line)[1], -1)
			// println("	Tags:", note.Tags)

		case rHeader.MatchString(line):
			block = block.backTo(0) // Header is the top level element
			html, err := renderHTML(EN_HEADER, rHeader.FindStringSubmatch(line)[1])
			if err != nil {
				return err
			}
			block.buffer_write(html)
			// println("	Header:", rHeader.FindStringSubmatch(line)[1])

		case rHr.MatchString(line):
			block = block.backTo(0) // Hr is the top level element
			block.buffer_write("\n<hr>\n")
			// println("	Hr:", line)

		case rPreCodeHead.MatchString(line):
			// println("	PreCode Start:", lines.num, line)
			son := block.spawn(EN_PRECODE)
			if err := parsePreCode(son, lines); err != nil {
				return err
			}
			block.buffer_write(son.html)
			// println("	PreCode End  :", lines.num, lines.current())

		case rBlockQuote.MatchString(line):
			// println("	Quote Start  :", lines.num, line)
			son := block.spawn(EN_BLOCKQUOTE)
			if err := parseBlockQuote(son, lines); err != nil {
				return err
			}
			block.buffer_write(son.html)
			// println("	Quote End    :", lines.num, lines.current())

		case rBulletList.MatchString(line):
			if err := parseBulletList(block, lines); err != nil {
				panic(err)
				return err
			}
		}
	}
	block.buffer_dump()
	for _, s := range block.children {
		println(s)
	}
	block.render()
	// println("\n++++++++++\nHTML:\n", block.html, "\n__________\n")
	note.Body = template.HTML(block.html)
	return nil
}

type Snippet struct {
	name ElemName
	text string
	elem []string
}

func renderSnippets(snippets []Snippet) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	for _, snip := range snippets {
		if snip.name == EN_TEXT {
			buf.WriteString(template.HTMLEscapeString(snip.text))
		} else {
			err := ELEMENTS_TMPL.ExecuteTemplate(buf, string(snip.name), snip.elem)
			if err != nil {
				return nil, err
			}
		}
	}
	// check BR
	// bytes.Contains(BR_FLAGS, subslice)
	if bytes.Contains(BR_FLAGS, buf.Bytes()[buf.Len()-1:]) {
		buf.WriteString("\n<br>\n")
	}
	return buf, nil
}

func inlineFilter(re *regexp.Regexp, name ElemName, text string) []Snippet {
	pos := 0
	snippets := []Snippet{}
	lastIdx := re.NumSubexp() * 2
	for _, i := range re.FindAllStringSubmatchIndex(text, -1) {
		if pos < i[0] {
			snippets = append(snippets, Snippet{name: EN_TEXT, text: text[pos:i[0]]})
		}
		matched := []string{}
		for n := 2; n <= lastIdx; n += 2 {
			if i[n] != -1 {
				matched = append(matched, text[i[n]:i[n+1]])
			}
		}
		snippets = append(snippets, Snippet{name: name, elem: matched})
		pos = i[1]
	}
	if pos < len(text) {
		snippets = append(snippets, Snippet{name: EN_TEXT, text: text[pos:]})
	}
	return snippets
}

func parseInLine(line string) (*bytes.Buffer, error) {
	reMap := map[ElemName]*regexp.Regexp{
		EN_CODE:   rCode,
		EN_IMG:    rImg,
		EN_LINK:   rLink,
		EN_STRONG: rStrong,
	}
	// filter inline elements
	snippets := []Snippet{Snippet{name: EN_TEXT, text: line}}
	for _, name := range []ElemName{EN_CODE, EN_IMG, EN_LINK, EN_STRONG} {
		re := reMap[name]
		temp := []Snippet{}
		for i := 0; i < len(snippets); i++ {
			if snippets[i].name == EN_TEXT {
				temp = inlineFilter(re, name, snippets[i].text)
				n := len(temp)
				temp = append(temp, snippets[i+1:]...)
				snippets = append(snippets[:i], temp...)
				i += n
			}
		}
	}
	// write into buffer
	if buf, err := renderSnippets(snippets); err != nil {
		return nil, err
	} else {
		return buf, nil
	}
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
		for cnt := lines.total(); lines.num < cnt; lines.num++ {
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
	html, err := renderHTML(EN_PRECODE, data)
	block.html = template.HTML(html)
	return err
}

func parseBlockQuote(block *Block, lines *Lines) error {
	var line string
	var text string
	for cnt := lines.total(); lines.num < cnt; lines.num++ {
		line = lines.current()
		if rBlockQuote.MatchString(line) {
			text = rBlockQuote.FindStringSubmatch(line)[1]
			text = strings.TrimSpace(text)
			if len(text) > 0 {
				block.buffer_write(text)
			} else {
				block.buffer_dump()
			}
		} else {
			break
		}
	}

	lines.jumpto(lines.num - 1) // back to the last line in BlockQuote
	return block.render()
}

func parseBulletLine(block *Block, line string) *Block {
	// Get level, elemname, text from line
	res := rBulletList.FindStringSubmatch(line)
	if len(res) != 3 {
		panic("Not a Bullet Line")
	}
	var name ElemName
	if strings.Contains("*+-", res[1]) {
		name = EN_UNORDERLIST
	} else {
		name = EN_ORDERLIST
	}
	text := res[2]
	level := getLevel(line, INDENT)

	// deal with the different level
	switch {
	case block.level < level: // son block level
		for block.level < level {
			block = block.spawn(name)
		}
		block.buffer_dump()
		block.buffer_write(text)

	case block.level == level:
		block.buffer_dump()
		block.buffer_write(text)

	case block.level > level: // father block level
		block = block.backTo(level)
	}
	return block
}

func parseBulletList(block *Block, lines *Lines) error {
	println("	Bullet Start :", lines.num, lines.current())
	var line string
	var blanks int
	ori_level := block.level
LOOP:
	for cnt := lines.total(); lines.num < cnt; lines.num++ {
		line = lines.current()
		switch {
		case rBulletList.MatchString(line):
			blanks = 0
			block = parseBulletLine(block, line)

		case rBlank.MatchString(line):
			block.buffer_dump()
			blanks++

		default:
			if blanks == 0 {
				block.buffer_write(line)
			} else {
				if err := block.render(); err != nil {
					return err
				}
				break LOOP
			}
			// reset blanks
			blanks = 0
		}

	}
	block = block.backTo(ori_level)

	lines.jumpto(lines.num - 1) // back to the last line of BulletList
	println("	Bullet End   :", lines.num, lines.current())

	return block.render()
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
	// _test()
	// line := "aaa`bbb`ccc`ddd`eee``fff``ggg``h`iii"
	line := "111`Miao`222**BBB**333[]()444![]()555."
	// re := rCode
	// fmt.Println(re.MatchString(line))

	// fmt.Println(re.FindAllStringSubmatchIndex(line, -1), re.NumSubexp())
	// res := inlineFilter(re, "----", line)
	// for i, snip := range res {
	// 	fmt.Println(i, ":", snip.name, snip.text)
	// }
	// fmt.Println(len(res))
	s, e := parseInLine(line)
	if e != nil {
		fmt.Println(e)
	} else {
		fmt.Printf("%T \n%v", s, s)
	}
}
