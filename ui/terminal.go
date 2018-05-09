package ui

import (
	"github.com/nsf/termbox-go"
	"github.com/mattn/go-runewidth"
)

type Terminal interface {
	Clear(fg, bg termbox.Attribute)
	Flush()
	HideCursor()
	Size() (int, int)
	SetCell(r rune, x, y int, fg, bg termbox.Attribute) int
}

// ---------------------------------------------------------------------------------------------------------------------

type terminal struct {}

func (t *terminal) Clear(fg, bg termbox.Attribute) {
	termbox.Clear(fg, bg)
}

func (t *terminal) Flush() {
	termbox.Flush()
}

func (t *terminal) HideCursor() {
	termbox.HideCursor()
}

func (t *terminal) Size() (int, int) {
	return termbox.Size()
}

func (t *terminal) SetCell(r rune, x, y int, fg, bg termbox.Attribute) int {
	termbox.SetCell(x, y, r, fg, bg)
	return runewidth.RuneWidth(r)
}

// ---------------------------------------------------------------------------------------------------------------------

type nullTerminal struct {}

func (t *nullTerminal) Clear(fg, bg termbox.Attribute) {}

func (t *nullTerminal) Flush() {}

func (t *nullTerminal) HideCursor() {}

func (t *nullTerminal) Size() (int, int) { return -1, -1 } // TODO: Should probably return something sensible here...

func (t *nullTerminal) SetCell(r rune, x, y int, fg, bg termbox.Attribute) int { return runewidth.RuneWidth(r) }
