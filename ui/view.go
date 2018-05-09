package ui

import (
	"github.com/nsf/termbox-go"
	"fmt"
	"github.com/g-dx/rosslyn/slack"
	"strings"
	"time"
	"hash/fnv"
	"strconv"
)

type View interface {
	OnKey(key termbox.Key, r rune)
	Draw(term Terminal)
}

// ---------------------------------------------------------------------------------------------------------------------

type ChannelSelectionView struct {
	ctrl Controller
	chls *ChannelList
	users *slack.UserList
	pos int
}

func NewChannelListView(ctrl Controller, chls *ChannelList, users *slack.UserList) *ChannelSelectionView {
	return &ChannelSelectionView{ ctrl: ctrl, chls: chls, users : users  }
}

func (csv *ChannelSelectionView) OnKey(key termbox.Key, r rune) {
	switch key {
	case termbox.KeyEnter:
		csv.ctrl.SwitchChannel(csv.chls.chls[csv.pos])
	case termbox.KeyArrowUp:
		csv.up()
	case termbox.KeyArrowDown:
		csv.down()
	case termbox.KeyEsc:
		csv.ctrl.SwitchChannel(nil)
	}
}

func (csv *ChannelSelectionView) up() {
	if csv.pos != 0 {
		csv.pos--
		csv.ctrl.Redraw()
	}
}

func (csv *ChannelSelectionView) down() {
	if csv.pos != csv.chls.Size() - 1 {
		csv.pos++
		csv.ctrl.Redraw()
	}
}

func (csv *ChannelSelectionView) Draw(term Terminal) {

	term.Clear(coldef, coldef)
	term.HideCursor()

	w, h := term.Size()
	printBorder(0, 0, w, h, term)
	printString("Channel List\n", 2, 1, termbox.ColorWhite | termbox.AttrUnderline, coldef, term)

	x, y := 1, 3
	for i, ch := range csv.chls.chls {
		bg := coldef
		fg := coldef

		if csv.pos == i {
			bg = termbox.ColorYellow
			fg = termbox.ColorWhite
		}

		unread := "    "
		if ch.unread > 0 {
			unread = fmt.Sprintf("%4v", fmt.Sprintf("(%v)", ch.unread))
		}

		// Check if this is a user
		var pos int
		if ch.id[:1] == "D" {
			switch csv.users.GetPresence(ch.user) {
			case "active":
				pos = printString("●", x, y, termbox.Attribute(30), coldef, term)
			case "away":
				pos = printString("○", x, y, termbox.ColorWhite, coldef, term)
			default:
				pos = printString("?", x, y, termbox.ColorRed, coldef, term)
			}
		} else {
			pos = printString("*", x, y, termbox.ColorYellow, coldef, term)
		}

		pos = printString(unread, pos+1, y, termbox.ColorWhite, coldef, term)
		printString(ch.name, pos+1, y, fg, bg, term)
		y++
	}
	term.Flush()
}

// ---------------------------------------------------------------------------------------------------------------------

type ChannelView struct {
	ctrl Controller

	cl *Channel
	msgLines []int

	editor EditBox
}

func NewChannelView(ctrl Controller, cl *Channel) *ChannelView {
	return &ChannelView{ ctrl: ctrl, cl: cl }
}

func (cv *ChannelView) OnKey(key termbox.Key, r rune) {
	switch key {

	case termbox.KeyPgup:
		cv.pageUp()
	case termbox.KeyPgdn:
		cv.pageDown()
	case termbox.KeyArrowUp:
		cv.up()
	case termbox.KeyArrowDown:
		cv.down()

	case termbox.KeyCtrlK:
		cv.ctrl.SelectChannel()

	case termbox.KeyEnter:
		// TODO: Should store this for upKey reedit scenario...
		text := cv.editor.GetText()
		cv.editor.Clear()
		cv.ctrl.SendMessage(text)

	// TODO: handle these better
	case termbox.KeyArrowLeft, termbox.KeyCtrlB:
		cv.editor.MoveCursorOneRuneBackward()
		cv.ctrl.Redraw()
	case termbox.KeyArrowRight, termbox.KeyCtrlF:
		cv.editor.MoveCursorOneRuneForward()
		cv.ctrl.Redraw()
	case termbox.KeyBackspace, termbox.KeyBackspace2:
		cv.editor.DeleteRuneBackward()
		cv.ctrl.Redraw()
	case termbox.KeyDelete, termbox.KeyCtrlD:
		cv.editor.DeleteRuneForward()
		cv.ctrl.Redraw()
	case termbox.KeyTab:
		cv.editor.InsertRune('\t')
		cv.ctrl.Redraw()
	case termbox.KeySpace:
		cv.editor.InsertRune(' ')
		cv.ctrl.Redraw()
	case termbox.KeyHome, termbox.KeyCtrlA:
		cv.editor.MoveCursorToBeginningOfTheLine()
		cv.ctrl.Redraw()
	case termbox.KeyEnd, termbox.KeyCtrlE:
		cv.editor.MoveCursorToEndOfTheLine()
		cv.ctrl.Redraw()
	default:
		cv.editor.InsertRune(r)
		cv.ctrl.Redraw()
	}
}

func (cv *ChannelView) pageUp() {
	cv.scroll(-10)
}

func (cv *ChannelView) pageDown() {
	cv.scroll(10)
}

func (cv *ChannelView) down() {
	cv.scroll(1)
}

func (cv *ChannelView) up() {
	cv.scroll(-1)
}

func (cv *ChannelView) scroll(inc int) {
	if cv.cl.pos + inc <= 0 {
		cv.cl.pos = 0
	} else if cv.cl.pos + inc >= len(cv.cl.msgs) {
		cv.cl.pos = len(cv.cl.msgs)-1
	} else {
		cv.cl.pos += inc
	}
	if cv.cl.pos <= 30 {
		cv.ctrl.LoadMessages(cv.cl)
	}
	cv.ctrl.Redraw()
}

const coldef = termbox.ColorDefault
const lineColour = termbox.ColorYellow

const monoFg = termbox.Attribute(197)
const monoBg = termbox.Attribute(239)

func (cv *ChannelView) Draw(term Terminal) {

	msgs := cv.cl.msgs
	msgPos := cv.cl.pos
	term.Clear(coldef, coldef)

	w, h := term.Size()
	ui.Printf("Terminal (w:%v, h:%v)\n", w, h)

	msgBoxHeight := h-4
	printBorder(0, 0, w, msgBoxHeight, term)

	var prev time.Time
	x, y := 1, msgBoxHeight-1

	for i := msgPos; len(msgs) > 0 && i >= 0; i-- {
		msg := msgs[i]

		// Print separator if day changes
		if i != len(msgs)-1 && prev.Day() > msg.T.Day() {
			y -= 2
			printString(buildSeparator(w, prev), x-1, y, lineColour, coldef, term)
			y--
		}
		prev = msg.T

		// Calculate required lines
		c := &canvas{w: w-2, h: h, term: &nullTerminal{}}
		drawMessage(msg, c)
		y -= c.Lines()
		drawMessage(msg, &canvas{x0: x, x: x,  y: y, w: w-2, h: h, term: term})
	}

	// Draw input box

	printBorder(0, msgBoxHeight, w, h-1, term)
	cv.editor.Draw(1, msgBoxHeight+1, w-2, 1)
	termbox.SetCursor(1+cv.editor.CursorX(), msgBoxHeight+1)

	// Draw status bar
	printString(formatUsersTyping(userTypingTimer.UsersTyping()), 1, h-1, coldef, coldef, term)

	term.Flush()
}

func RequiredLines(pos0, w int, msg []rune) int {

	c := &canvas{x0: pos0, w: w, h: 1 << 32, term: &nullTerminal{}}
	c.Printf(msg, coldef, coldef)
	return c.Lines()
}

// TODO: Return []Style here too so that we colour these appropriately?
func formatUsersTyping(users []string) string {
	switch len(users) {
	case 0:
		return ""
	default:
		conj := "is"
		if len(users) > 1 {
			conj = "are"
		}
		return fmt.Sprintf("%v %v typing...", strings.Join(users, ","), conj)
	}
}

func buildSeparator(w int, t time.Time) string {
	ts := t.Format("Jan 2")
	return fmt.Sprintf("├%v %v ─┤", strings.Repeat("─", w-5-len(ts)), ts)
}

func ColoursForFormat(rs []rune, f fmtType) (termbox.Attribute, termbox.Attribute) {
	switch f {
	case _Channel:
		return termbox.Attribute(118), coldef
	case User:
		return getColour(string(rs[1:])), coldef
	case Monospaced, Preformatted:
		return monoFg, monoBg
	case Variable:
		return termbox.ColorBlack, termbox.ColorYellow
	case Emoji:
		return termbox.Attribute(227), coldef
	case Link:
		return coldef | termbox.AttrUnderline, coldef
	default:
		return coldef, coldef
	}
}


func getColour(s string) termbox.Attribute {
	colours := []termbox.Attribute { termbox.ColorRed, termbox.ColorBlue, termbox.ColorGreen,
					 termbox.ColorCyan, termbox.Attribute(227), termbox.Attribute(200), termbox.Attribute(171)}
	h := fnv.New32a()
	h.Write([]byte(s))
	return colours[int(h.Sum32()) % len(colours)]
}

func tsToTime(ts string) time.Time {
	i, err := strconv.ParseInt(ts[:strings.Index(ts, ".")], 10, 64)
	if err != nil {
		panic(err)
	}
	return time.Unix(i, 0)
}

func parseTimestamp(t time.Time) string {
	ts := t.Format("3:04 PM") // Uses current locale/timezone by default
	if len(ts) == 7 {
		ts = " " + ts
	}
	return ts
}


func printBorder(x, y, w, h int, term Terminal) {
	term.SetCell('╭', x, y, lineColour, coldef)
	term.SetCell('╮', w-1, y, lineColour, coldef)
	term.SetCell( '╰', x, h-1, lineColour, coldef)
	term.SetCell('╯', w-1, h-1, lineColour, coldef)
	for i := x+1; i < w-1 ; i++ {
		term.SetCell('─', i, y, lineColour, coldef)
		term.SetCell('─', i, h-1, lineColour, coldef)
	}
	for i := y+1; i < h-1 ; i++ {
		term.SetCell('│', x, i, lineColour, coldef)
		term.SetCell('│', w-1, i, lineColour, coldef)
	}
}

func printString(s string, x, y int, fg, bg termbox.Attribute, term Terminal) int {
	for _, r := range s {
		x += term.SetCell(r, x, y, fg, bg)
	}
	return x
}

func drawMessage(msg *Message, c Canvas) {

	// Print message prefix
	c.Printsf(parseTimestamp(msg.T), coldef, coldef)
	c.Move(1, 0)
	c.Printsf(msg.User, getColour(msg.User), coldef)
	c.Move(1, 0)
	if msg.IsEdited {
		c.Printsf("(edited)", termbox.ColorBlue, coldef)
		c.Move(1, 0)
	}

	// Print message content
	defFg := coldef
	defBg := coldef
	pos := 0
	if len(msg.Formats) > 0 {
		rs := []rune(msg.Text)
		for _, format := range msg.Formats {
			if format.Start() > pos {
				c.Printf(rs[pos:format.Start()], defFg, defBg)
			}

			// Get styling & determine colours
			styledText := rs[format.Start():format.End()]
			fg, bg := ColoursForFormat(styledText, format.Type())

			// Preformatted text has background across whole width
			if format.Type() == Preformatted {
				x, _ := c.Position()
				w, _ := c.Size()
				c.Background(x, w-x, RequiredLines(x, w-x, styledText)-1, bg)
			}

			// Print with styling
			c.Printf(styledText, fg, bg)
			pos = format.End()
		}
		// Print remaining (if any) text
		c.Printf(rs[pos:], defFg, defBg)
	} else {
		c.Printsf(msg.Text, defFg, defBg)
	}
}