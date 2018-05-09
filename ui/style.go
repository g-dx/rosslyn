package ui

import (
	"fmt"
	"regexp"
	"unicode"
)

//
// ---------------------------------------------------
// | type (4 bits) | End (16 bits) | Start (16 bits) |
// ---------------------------------------------------
//
type format int64

func NewFormat(start, end int, fType fmtType) format {
	f := 0
	f |= start
	f |= end << 16
	f |= int(fType) << 32
	return format(f)
}

func (f format) Start() int {
	return int(0xFFFF & f)
}

func (f format) End() int {
	return int(0xFFFF0000 &f) >> 16
}

func (f format) Type() fmtType {
	return fmtType((0xF00000000 & f) >> 32)
}

func (f format) String() string {
	return fmt.Sprintf("%v(%v,%v)", f.Type(), f.Start(), f.End())
}

type fmtType byte

const (
	// NOTE: Currently represented by 4-bits so max of 16 values!
	Bold          fmtType = iota
	Strikethrough
	Underlined
	Italic
	Monospaced
	Preformatted
	_Channel
	User
	Variable
	Emoji
	Link
	Unknown
)

// TODO: Maybe consider stringer instead? https://godoc.org/golang.org/x/tools/cmd/stringer
func (f fmtType) String() string {
	switch f {
	case Bold:
		return "Bold"
	case Strikethrough:
		return "Strikethrough"
	case Underlined:
		return "Underlined"
	case Italic:
		return "Italic"
	case Monospaced:
		return "Monospaced"
	case Preformatted:
		return "Preformatted"
	case _Channel:
		return "Channel"
	case User:
		return "User"
	case Variable:
		return "Variable"
	case Emoji:
		return "Emoji"
	case Link:
		return "Link"
	case Unknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

type Lookup interface {
	GetUser(user string) string
	GetChannel(channel string) string
}

const eof = -1
var preformattedSuffix = []rune{'`', '`', '`'}
var preformattedPrefix = []rune{'`', '`'}
var slackSeqRegex *regexp.Regexp
var emojiRegex *regexp.Regexp

func init() {
	slackSeqRegex = regexp.MustCompile("^([#!@])([^|]+)\\|?(.+)?$")
	emojiRegex = regexp.MustCompile("^[+_a-z0-9]+(?:[+_a-z0-9]+)*$")
}

type Formatter struct {
	content []rune
	formats []format
	buf     []rune
	pos     int
	lookup  Lookup
}

func (fe *Formatter) Format(s string) ([]rune, []format) {

	// Reset
	fe.buf = []rune(s)
	fe.content = fe.content[:0]
	fe.formats = fe.formats[:0]
	fe.pos = 0

done:
	for {
		switch r := fe.Next(); {
		case r == '<':
			fe.slackFormatting()
		case r == '`':
			fe.monoSpaced()
		case r == '&':
			fe.escapeSequence()
		case r == ':':
			fe.emoji()
		case isBasicMarkdown(r) && isAlphanumeric(fe.Peek()):
			fe.basicMarkdown(r)
		case r == eof:
			break done
		default:
			fe.AddRune(r)
		}
	}
	return fe.content, fe.formats
}

func isBasicMarkdown(r rune) bool {
	return r == '_' || r == '*' || r == '~'
}

func isAlphanumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (fe *Formatter) Peek() rune {
	if fe.pos == len(fe.buf) {
		return eof
	}
	return fe.buf[fe.pos]
}

func (fe *Formatter) emoji() {
	// Found possible emoji run
	if ok, pos := fe.Find(':'); ok {
		seq := fe.buf[fe.pos:pos]
		if !emojiRegex.MatchString(string(seq)) {
			fe.AddRune(':')
			return // Not a valid emoji
		}

		// TODO: Move this mapping elsewhere
		switch string(seq) {
		case "slightly_smiling_face":
			fe.AddFormattedRunes([]rune {'😊' }, Emoji)
		case "pizza":
			fe.AddFormattedRunes([]rune {'🍕' }, Emoji)
		default:
			fe.AddFormattedRunes(seq, Emoji)
		}

		fe.Discard(len(seq) + 1)
		return
	}

	// Just a lone colon...
	fe.AddRune(':')
}

func (fe *Formatter) basicMarkdown(r rune) {

	// Found markdown run
	if ok, pos := fe.Find(r); ok {
		seq := fe.buf[fe.pos:pos]
		fe.AddFormattedRunes(seq, markupToFormat(r))
		fe.Discard(len(seq) + 1)
		return
	}

	// Just a lone markup character...
	fe.AddRune(r)
}

func (fe *Formatter) monoSpaced() {

	// Check for preformatted run
	if fe.PeekRun(preformattedPrefix) {
		ok, pos := fe.FindRun(preformattedSuffix)
		if !ok {
			fe.AddRune('`')
			return // Preformatted block unclosed
		}
		fe.Discard(2)

		preformat := fe.buf[fe.pos:pos]

		// Output a newline if previous character wasn't a newline or the next one isn't either
		if len(fe.content) == 0 || (len(fe.content) > 0 && fe.content[len(fe.content)-1] != '\n') && fe.Peek() != '\n' {
			fe.AddRune('\n')
		}
		fe.AddFormattedRunes(preformat, Preformatted)
		fe.Discard(len(preformat) + 3)

		// Output newline if last character wasn't a newline or the next one isn't
		if (len(fe.content) > 0 && fe.content[len(fe.content)-1] != '\n') && fe.Peek() != '\n' {
			fe.AddRune('\n')
		}
		return
	}

	// Found monospaced run
	if ok, pos := fe.Find('`'); ok {
		mono := fe.buf[fe.pos:pos]
		fe.AddFormattedRunes(mono, Monospaced)
		fe.Discard(len(mono) + 1)
		return
	}

	// Just a lone backtick...
	fe.AddRune('`')
}

func (fe *Formatter) escapeSequence() {

	// Found escape sequence
	if ok, pos := fe.Find(';'); ok {
		seq := fe.buf[fe.pos:pos]
		switch string(seq) {
		case "amp":
			fe.AddRune('&')
		case "lt":
			fe.AddRune('<')
		case "gt":
			fe.AddRune('>')
		default:
			// Unknown escape sequence - output as is
			fe.AddRune('&')
			fe.AddRunes(seq)
			fe.AddRune(';')
		}
		fe.Discard(len(seq) + 1)
		return
	}

	// Just a lone ampersand...
	// TODO: Technically this is a violation of the Slack specification.
	fe.AddRune('&')
}

func (fe *Formatter) slackFormatting() {
	// TODO: This is not complete!
	// https://api.slack.com/docs/message-formatting#how_to_display_formatted_messages
	if ok, pos := fe.Find('>'); ok {
		seq := fe.buf[fe.pos:pos]
		parts := slackSeqRegex.FindStringSubmatch(string(seq))
		switch len(parts) {
		case 4:
			// Type + Value + Label (possibly empty)
			format := slackTypeToFormat(parts[1])
			label := []rune(parts[3])

			// Check if label is empty - perform lookup
			if len(label) == 0 {
				label = []rune(fe.Lookup(format, parts[2]))
			} else if format == _Channel && label[0] != '#' {
				label = append([]rune{'#'}, label...) // For some reason the slack client outputs <#Cxxxx|channel> without a '#' ...?
			}
			fe.AddFormattedRunes(label, format)
		case 0:
			if len(seq) > 0 {
				//label := ""
				//val := string(seq)
				//if strings.ContainsRune(val, '|') {
				//	split := strings.Split(string(seq), "|")
				//	val = split[0]
				//	label = split[1]
				//}
				//fe.AddRune('\n')
				//fe.AddRune('🔗')
				//fe.AddFormattedRunes([]rune(label), Emoji)
				//fe.AddRunes([]rune(fmt.Sprintf("\n╭%v╮\n", strings.Repeat("─", len(val)))))
				//fe.AddRune('│')
				//fe.AddFormattedRunes([]rune(val), Link)
				//fe.AddRune('|')
				//fe.AddRunes([]rune(fmt.Sprintf("\n╰%v╯\n", strings.Repeat("─", len(val)))))
				fe.AddRune('🔗')
				fe.AddFormattedRunes(seq, Link)
			}
		}
		fe.Discard(len(seq) + 1)
		return
	}

	// Just a lone less than...
	fe.AddRune('<')
}

func markupToFormat(r rune) fmtType {
	switch r {
	case '*':
		return Bold
	case '_':
		return Italic
	case '~':
		return Strikethrough
	default:
		return Unknown
	}
}

func slackTypeToFormat(slackType string) fmtType {
	switch slackType {
	case "@":
		return User
	case "#":
		return _Channel
	case "!":
		return Variable
	default:
		return Unknown
	}
}

func (fe *Formatter) Next() rune {
	if fe.pos == len(fe.buf) {
		return eof
	}
	r := fe.buf[fe.pos]
	fe.pos++
	return r
}

func (fe *Formatter) Discard(n int) {
	fe.pos += n
}

func (fe *Formatter) Lookup(f fmtType, value string) string {
	switch f {
	case User:
		return "@" + fe.lookup.GetUser(value)
	case _Channel:
		return "#" + fe.lookup.GetChannel(value)
	case Variable:
		return value
	}
	return value
}

func (fe *Formatter) PeekRun(run []rune) bool {
	for i := 0; i < len(run); i++ {
		if i+fe.pos == len(fe.buf) {
			return false // EOF
		}
		if fe.buf[i+fe.pos] != run[i] {
			return false // No match
		}
	}
	return true
}

func (fe *Formatter) FindRun(run []rune) (bool, int) {

outer:
	for pos := fe.pos; pos < len(fe.buf); pos++ {
		if fe.buf[pos] == run[0] {
			for i := 0; i < len(run); i++ {
				if i+pos == len(fe.buf) {
					return false, -1 // EOF
				}
				if fe.buf[i+pos] != run[i] {
					continue outer
				}
			}
			return true, pos
		}
	}
	return false, -1
}

func (fe *Formatter) Find(r rune) (bool, int) {
	for i := fe.pos; i < len(fe.buf); i++ {
		if fe.buf[i] == r {
			return true, i
		}
	}
	return false, -1
}

func (fe *Formatter) AddRune(r rune) {
	fe.content = append(fe.content, r)
}

func (fe *Formatter) AddRunes(rs []rune) {
	fe.content = append(fe.content, rs...)
}

func (fe *Formatter) AddFormattedRunes(rs []rune, f fmtType) {
	fe.formats = append(fe.formats, NewFormat(len(fe.content), len(fe.content)+len(rs), f))
	fe.AddRunes(rs)

}
