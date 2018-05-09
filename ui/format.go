package ui

import (
	"github.com/nsf/termbox-go"
	"strings"
)

/*

 TODO: The whole implementation here can be simplified as there is (probably) no need:
  -- To buffer characters anymore
  -- Store default colours

 */

/**
 Rules for line breaking.

  1. Loop over []rune printing each one and xc++
  2. Every time a ' ' rune calculate `wLen` which is the number of runes until next ' ' or EOL
  3.
    -- If wLen + xc < w - print word
    -- If wLen + xc >= w && wLen > w
    -- If wLen + xc >=


 */
type Canvas interface {
	NewLine()
	Background(x, w, n int, bg termbox.Attribute)
	Printf(rs []rune, fg, bg termbox.Attribute)
	Printsf(s string, fg, bg termbox.Attribute)
	Flush()
	Position() (int, int)
	Size() (int, int)
	Move(x, y int)
	Lines() int
}

type canvas struct {
	x0, x, y, w, h int
	buf         []rune
	fg, bg      termbox.Attribute
	term        Terminal
}

func NewCanvas(term Terminal) Canvas {
	return &canvas{term: term}
}

func (c *canvas) Background(x, w, n int, bg termbox.Attribute) {
	for i := n; i >= 0; i-- {
		c.Printf([]rune(strings.Repeat(" ", w)), coldef, bg)
	}
	c.y -= n+1
}

func (c *canvas) Flush() {
	c.printBuf()
}

func (c *canvas) NewLine() {
	c.y++
	c.x = c.x0
}

func (c *canvas) Move(x, y int) {
	c.x += x
	c.y += y
}

func (c *canvas) Size() (int, int) {
	return c.w, c.h
}

func (c *canvas) Printsf(s string, fg, bg termbox.Attribute) {
	c.Printf([]rune(s), fg, bg)
}

func (c *canvas) Printf(rs []rune, fg, bg termbox.Attribute) {

	ui.Printf("x:%v,y:%v,w:%v,h:%v - %v\n", c.x, c.y, c.w, c.h, string(rs))
	c.SetColours(fg, bg)
	for _, r := range rs {
		c.Print(r)
	}
	c.Flush()
}

func (c *canvas) Print(r rune) {

	// Buffer all non-space runes
	switch r {
	case '\n':
		// Check if current buffer will fit on existing line
		if c.x + len(c.buf) < c.w { // TODO: width is bytes & buf is runes - they might not always match
			c.printBuf()
		}
		c.NewLine()
	case ' ':
		c.printBuf()
		c.printImpl(r)
	default:
		//debug.Printf("Word size: %v\n", len(c.buf))
		if len(c.buf) + 1 == c.w {
			// Buffer contains a string with no spaces which is the width of the screen
			//debug.Printf("Very long line: w:%v, x:%v, buf: %v\n", c.w, c.x, string(c.buf))
			c.printBuf()
			c.NewLine()
		}
		c.buf = append(c.buf, r)
	}
}

func (c *canvas) printImpl(r rune) {
	c.x += c.term.SetCell(r, c.x, c.y, c.fg, c.bg)
}

func (c *canvas) printBuf() {
	//debug.Printf("Printing: %v\n", string(c.buf))
	if c.x + len(c.buf) >= c.w { // TODO: width is bytes & buf is runes - they might not always match
		c.NewLine()
	}
	for _, r := range c.buf {
		c.printImpl(r)
	}
	c.buf = c.buf[:0]
}

func (c *canvas) SetColours(fg, bg termbox.Attribute) {
	c.Flush() // Print existing content before changing colour
	c.fg = fg
	c.bg = bg
}

func (c *canvas) Position() (int, int) {
	return c.x, c.y
}

func (c canvas) Lines() int {
	return c.y + 1
}
