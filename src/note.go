package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ElemName string

const (
	INDENT = 4 // indent spaces in note files

	// Element names
	EN_ROOT        ElemName = "Note"
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
	// others
	EN_TEXT ElemName = "_text"
	EN_HTML ElemName = "_html"
)

var (
	// Custom elements
	rWeather = regexp.MustCompile(`^Weather: {1,3}(\S[\S ]*)`) // Weather: 🌞 ⛅
	rAuth    = regexp.MustCompile(`^Auth: {1,3}(\S[\S ]*)`)    // Auth: Lee Bai
	rTags    = regexp.MustCompile(`^Tags: {1,3}(\S[\S ]*)`)    // Tags: name1 name2 name3
	// Standard elements
	rHeader      = regexp.MustCompile(`^#{1,6} {1,3}(\S[\S ]*)`)
	rHr          = regexp.MustCompile(`^(-{3,}|_{3,}|\*{3,})\s*$`)
	rBulletList  = regexp.MustCompile(`^[ \t]*([*+-]|\d+\.) {1,3}(\S[\S ]*\s*$)`)
	rBlockQuote  = regexp.MustCompile(`^[ \t]*> {0,3}(\S[\S ]*|\s*$)`)
	rPreCodeHead = regexp.MustCompile("(?:^[ \t]*)``` {0,3}(\\w*)")
	rPreCodeTail = regexp.MustCompile("(?:^[ \t]*)```\\s*$")
	rCode        = regexp.MustCompile("``(.+?)``|`(.+?)`")
	rStrong      = regexp.MustCompile(`\*\*(.+?)\*\*`)
	rImg         = regexp.MustCompile(`!\[([\S ]*?)\]\((\S*?)(?: \S*?)*?\)`)
	rLink        = regexp.MustCompile(`\[([\S ]*?)\]\((\S*?)(?: \S*?)*?\)`)
	// Others
	rDate  = regexp.MustCompile(`\d{4}-(0[1-9]|1[0-2])-([0-2]\d)|3[01]`)
	rWord  = regexp.MustCompile(`\S+`)
	rBlank = regexp.MustCompile(`^\s*$`)

	BR_FLAGS = []byte(`.。"”!！?？…`) // symbols for line break

	ELEMENTS_TMPL *template.Template // have to initialized by func init()
	Funcs         = template.FuncMap{
		"safe":   HTML,
		"inline": parseInline,
	}

	InlineRegexps = map[ElemName]*regexp.Regexp{
		EN_IMG:    rImg,
		EN_LINK:   rLink,
		EN_CODE:   rCode,
		EN_STRONG: rStrong,
	}
)

func init() {
	// init ELEMENTS_TMPL
	tmplPath := filepath.Join(RunningDir(), "elements.tmpl")
	tmpl, err := template.New("").Funcs(Funcs).ParseFiles(tmplPath)
	ELEMENTS_TMPL = template.Must(tmpl, err)
}

func HTML(text interface{}) template.HTML {
	switch value := text.(type) {
	case []byte:
		return template.HTML(value)
	case string:
		return template.HTML(value)
	case *bytes.Buffer:
		return template.HTML(value.Bytes())
	case template.HTML:
		return value
	default:
		return template.HTML("")
	}
}

type Lines struct {
	num     int
	blanks  []int
	content []string
}

func (l *Lines) read(path string) error {
	if bytes, err := ioutil.ReadFile(path); err != nil {
		return err
	} else {
		l.content = strings.SplitAfter(string(bytes), "\n")
		return nil
	}
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

func (l *Lines) isBlank(n int) bool {
	for _, i := range l.blanks {
		if i == n {
			return true
		}
	}
	return false
}

func (l *Lines) setBlank(n int) bool {
	if l.isBlank(n) {
		return false
	} else {
		l.blanks = append(l.blanks, n)
		return false
	}
}

func (l *Lines) backwardBlanks(num int) int {
	if !sort.IntsAreSorted(l.blanks) {
		sort.Ints(l.blanks)
	}

	cb := 0
	idx := sort.SearchInts(l.blanks, num)
	if idx < len(l.blanks) {
		for num >= 0 && idx >= 0 && l.blanks[idx] == num {
			cb++
			num--
			idx--
		}
	}

	return cb
}

type Snippet struct {
	name ElemName
	text string
	elem []string
}

func snippetsAppend(snippets []Snippet, snip Snippet) []Snippet {
	if n := len(snippets); n > 0 {

		if snippets[n-1].name == EN_TEXT && snip.name == EN_TEXT {
			snippets[n-1].text += snip.text
		} else {
			snippets = append(snippets, snip)
		}
	} else {
		snippets = append(snippets, snip)
	}
	return snippets
}

func snippetsRender(snippets []Snippet) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	for _, snip := range snippets {
		if snip.name == EN_HTML {
			buf.WriteString(snip.text)
		} else if snip.name == EN_TEXT {
			buf.WriteString(template.HTMLEscapeString(snip.text))
		} else {
			err := ELEMENTS_TMPL.ExecuteTemplate(buf, string(snip.name), snip.elem)
			if err != nil {
				return nil, err
			}
		}
	}
	// TODO: add <br> when find BR_FLAGS at the tail of line
	// if bytes.Contains(BR_FLAGS, buf.Bytes()[buf.Len()-1:]) {
	// 	buf.WriteString("\n<br>\n")
	// }
	return buf, nil
}

type Block struct {
	name  ElemName
	level int

	children []string
	snippets []Snippet

	html template.HTML
	prev *Block
}

func (b *Block) spawn(name ElemName) *Block {
	return &Block{name: name, level: b.level + 1, prev: b}
}

func (b *Block) update(content interface{}) (err error) {
	switch value := content.(type) {
	case string:
		snip := Snippet{name: EN_TEXT, text: value}
		b.snippets = snippetsAppend(b.snippets, snip)
	case template.HTML:
		snip := Snippet{name: EN_HTML, text: string(value)}
		b.snippets = snippetsAppend(b.snippets, snip)
	case Snippet:
		b.snippets = snippetsAppend(b.snippets, value)
	case []Snippet:
		b.snippets = append(b.snippets, value...)
	}
	return err
}

func (b *Block) dump() {
	if len(b.snippets) > 0 {
		buf, err := parseInline(b.snippets)
		if err != nil {
			panic(err)
		}
		b.children = append(b.children, buf.String())
	}
	b.snippets = b.snippets[:0]
}

func (b *Block) render() error {
	b.dump()
	if len(b.children) > 0 {
		buf := new(bytes.Buffer)
		err := ELEMENTS_TMPL.ExecuteTemplate(buf, string(b.name), b.children)
		if err != nil {
			return err
		} else {
			b.html = template.HTML(buf.String())
		}
	}
	return nil
}

func (b *Block) backTo(level int) *Block {
	block := b
	if level >= 0 {
		for ; block.level > level; block = block.prev {
			if err := block.render(); err != nil {
				panic(err)
			}
			block.prev.update(block.html)
			block.prev.dump()
		}
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

type Note struct {
	Title   string
	Date    string
	Auth    string
	Weather string
	Tags    []string
	Body    template.HTML

	Slug    string
	Summary string // TODO: implement Summary
}

// Parse a *.note file into note.Body
func (note *Note) ParseFile(path string) error {
	// parse the filename
	if err := note.parsePath(path); err != nil {
		return err
	}
	// read the file
	lines := new(Lines)
	if err := lines.read(path); err != nil {
		return err
	}

	// parse *.note file
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
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	r := regexp.MustCompile(`[^\w\d\p{L}\p{N}]+`)
	return r.ReplaceAllString(name, "_")
}

func (note *Note) parseContent(lines *Lines) error {
	var line string
	block := &Block{name: EN_ROOT, level: 0}

	for cnt := lines.total(); lines.num < cnt; lines.num++ {
		line = lines.current()

		switch {
		case lines.num == 0:
			note.Title = strings.TrimSpace(line)

		case rWeather.MatchString(line):
			note.Weather = rWeather.FindStringSubmatch(line)[1]

		case rAuth.MatchString(line):
			note.Auth = strings.TrimSpace(rAuth.FindStringSubmatch(line)[1])

		case rTags.MatchString(line):
			note.Tags = rWord.FindAllString(rTags.FindStringSubmatch(line)[1], -1)

		case rHeader.MatchString(line):
			block = block.backTo(0).spawn(EN_HEADER)
			block.update(rHeader.FindStringSubmatch(line)[1])
			block = block.backTo(0)

		case rHr.MatchString(line):
			block = block.backTo(0) // Hr is the top level element
			block.update(template.HTML("<hr>\n"))
			block.dump()

		case rPreCodeHead.MatchString(line):
			son := block.spawn(EN_PRECODE)
			if err := parsePreCode(son, lines); err != nil {
				return err
			}
			block.update(son.html)

		case rBlockQuote.MatchString(line):
			son := block.spawn(EN_BLOCKQUOTE)
			if err := parseBlockQuote(son, lines); err != nil {
				return err
			}
			block.update(son.html)

		case rBulletList.MatchString(line):
			if err := parseBulletList(block, lines); err != nil {
				return err
			}

		case rBlank.MatchString(line):
			lines.setBlank(lines.num)
			block.dump()
			block = block.backTo(block.level - 1)

		default:
			if block.name != EN_PARAGRAPH {
				block = block.spawn(EN_PARAGRAPH)
			}
			block.update(line)
		}
	}

	block.dump()
	block.render()
	note.Body = block.html
	return nil
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

func parseInline(line interface{}) (*bytes.Buffer, error) {
	// type assertion for line
	var snippets []Snippet
	switch value := line.(type) {
	case string:
		snippets = []Snippet{Snippet{name: EN_TEXT, text: value}}
	case []byte:
		snippets = []Snippet{Snippet{name: EN_TEXT, text: string(value)}}
	case []Snippet:
		snippets = value
	}
	// filter inline elements
	for _, name := range []ElemName{EN_CODE, EN_IMG, EN_LINK, EN_STRONG} {
		re := InlineRegexps[name]
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
	if buf, err := snippetsRender(snippets); err != nil {
		return nil, err
	} else {
		return buf, nil
	}
}

func parsePreCode(block *Block, lines *Lines) error {
	// check first line
	line := lines.current()
	if !rPreCodeHead.MatchString(line) {
		return errors.New("Not match PreCode")
	}

	// get the lang
	lang := rPreCodeHead.FindStringSubmatch(line)[1]
	block.children = append(block.children, lang)
	lines.num++

	// pick out the codes
	codes := new(bytes.Buffer)
	for cnt := lines.total(); lines.num < cnt; lines.num++ {
		line = lines.current()
		if rPreCodeTail.MatchString(line) {
			break
		} else {
			codes.WriteString(line)
		}
	}

	block.children = append(block.children, codes.String())
	return block.render()
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
				block.update(text)
			} else {
				block.dump()
			}
		} else {
			break
		}
	}

	lines.jumpto(lines.num - 1) // back to the last line in BlockQuote
	return block.render()
}

func parseBulletLine(block *Block, line string) *Block {
	// Get the `level` `elemname` and `text` from line
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

	// change block with level same as the line
	if block.level < level {
		// son block level
		for block.level < level {
			block = block.spawn(name)
		}
	} else if block.level > level {
		// father block level
		block = block.backTo(level)
	}

	// deal with the text
	block.dump()
	block.update(text)

	return block
}

func parseBulletList(block *Block, lines *Lines) error {
	var line string
	ori_level := block.level
LOOP:
	for cnt := lines.total(); lines.num < cnt; lines.num++ {
		line = lines.current()
		switch {
		case rBulletList.MatchString(line):
			block = parseBulletLine(block, line)

		case rBlank.MatchString(line):
			block.dump()
			lines.setBlank(lines.num)

		default:
			blanks := lines.backwardBlanks(lines.num - 1)
			if blanks == 0 {
				block.update(line)
			} else {
				break LOOP
			}
		}
	}
	block = block.backTo(ori_level)
	block.dump()

	lines.jumpto(lines.num - 1) // back to the last line of BulletList
	return block.render()
}

// test code
//
// func _test() {
// 	note := new(Note)
// 	if err := note.ParseFile("../_test/2017-04-09_演示文稿.note"); err != nil {
// 		fmt.Println(err)
// 	} else {
// 		fmt.Printf("<div><h1>%v</h1></div>\n", note.Title)
// 		fmt.Printf("<div>日期：%v<br>天气：%v</div>\n", note.Date, note.Weather)
// 		fmt.Printf("<div>——%v</div>\n", note.Auth)
// 		fmt.Println(note.Body)
// 	}
// }

// func main() {
// 	_test()
// }
