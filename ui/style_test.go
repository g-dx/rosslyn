package ui

import (
	"reflect"
	"testing"
)

const errorString = "%-10v(%v)\nGot : '%v'\nWant: '%v'"

func TestStyle(t *testing.T) {
	tests := []struct {
		s          format
		start, end int
		f          fmtType
	}{
		{NewFormat(0, 0, Bold), 0, 0, Bold},
		{NewFormat(65535, 65535, Bold), 65535, 65535, Bold},
		{NewFormat(0, 1, Bold), 0, 1, Bold},
		{NewFormat(0, 1, Bold), 0, 1, Bold},
		{NewFormat(0, 1, Italic), 0, 1, Italic},
		{NewFormat(0, 1, Strikethrough), 0, 1, Strikethrough},
		{NewFormat(0, 1, Monospaced), 0, 1, Monospaced},
		{NewFormat(0, 1, Preformatted), 0, 1, Preformatted},
		{NewFormat(0, 1, _Channel), 0, 1, _Channel},
		{NewFormat(0, 1, User), 0, 1, User},
		{NewFormat(0, 1, Variable), 0, 1, Variable},
	}

	for _, data := range tests {
		if data.s.Start() != data.start {
			t.Errorf("Start: Got: %v, Wanted: %v", data.s.Start(), data.start)
		}
		if data.s.End() != data.end {
			t.Errorf("End: Got: %v, Wanted: %v", data.s.End(), data.end)
		}
		if data.s.Type() != data.f {
			t.Errorf("Format: Got: %v, Wanted: %v", data.s.Type(), data.f)
		}
	}
}

func TestFormatExtractor(t *testing.T) {

	tests := []struct {
		s          string
		content    string
		formatting []format
	}{
		 //No formatting
		{"", "", nil },
		{"simple", "simple", nil},
		{"simple multiple words", "simple multiple words", nil},

		// Escapes
		{"&amp;", "&", nil},
		{"&lt;", "<", nil},
		{"&gt;", ">", nil},
		{"&amp;&lt;&gt;", "&<>", nil},
		{"&amp;amp;", "&amp;", nil},
		{"&unknown;", "&unknown;", nil},

		// Monospaced & preformatted
		{"`mono`", "mono", fmts(NewFormat(0, 4, Monospaced))},
		{"```preformat```", "\npreformat\n", fmts(NewFormat(1, 10, Preformatted))},
		{"\n```preformat```", "\npreformat\n", fmts(NewFormat(1, 10, Preformatted))},
		{"\n```preformat```\n", "\npreformat\n", fmts(NewFormat(1, 10, Preformatted))},
		{"`mono````preformat```", "mono\npreformat\n",
		 fmts(NewFormat(0, 4, Monospaced), NewFormat(5, 14, Preformatted))},
		{"`*markdown*<@slack|format>:emoji:`", "*markdown*<@slack|format>:emoji:", fmts(NewFormat(0, 32, Monospaced))},
		// TODO: Fix escaping in monospacing/preformatting and enable these tests
		//{"`&amp; &lt; &gt;`", "& < >", fmts(NewFormat(0, 5, Monospaced))},
		//{"```&amp; &lt; &gt;```", "& < >", fmts(NewFormat(0, 5, Preformatted))},
		{"`", "`", make([]format, 0)},
		{"``", "", fmts(NewFormat(0, 0, Monospaced))}, // TODO: Is this valid?
		//{"```", "```", make([]format, 0) }, // TODO: Is this valid?

		// Slack escapes
		{"<@id>", "@user", fmts(NewFormat(0, 5, User)) },
		{"<#id>", "#channel", fmts(NewFormat(0, 8, _Channel)) },
		{"<@id|user>", "user", fmts(NewFormat(0, 4, User))},
		{"<#id|channel>", "#channel", fmts(NewFormat(0, 8, _Channel))},
		{"<#id|#channel>", "#channel", fmts(NewFormat(0, 8, _Channel))},
		{"<!id|variable>", "variable", fmts(NewFormat(0, 8, Variable))},
		{"<link>", "🔗link", fmts(NewFormat(1, 5, Link))},
		//{"<link|label>", "[label]:link", fmts(NewFormat(0, 4, Link))}, // TODO: Implement me!
		{"<", "<", make([]format, 0)},
		{"<>", "", make([]format, 0)},

		// Markdown
		{"*Bold*", "Bold", fmts(NewFormat(0, 4, Bold))},
		{"*Bo ld  *", "Bo ld  ", fmts(NewFormat(0, 7, Bold))},
		{"_Italic_", "Italic", fmts(NewFormat(0, 6, Italic))},
		{"_Ita lic  _", "Ita lic  ", fmts(NewFormat(0, 9, Italic))},
		{"~Strikethrough~", "Strikethrough", fmts(NewFormat(0, 13, Strikethrough))},
		{"~Strike through ~", "Strike through ", fmts(NewFormat(0, 15, Strikethrough))},
		{"* no * _ formatting _ ~ applied ~", "* no * _ formatting _ ~ applied ~", make([]format, 0)},
		{"`* mono *`", "* mono *", fmts(NewFormat(0, 8, Monospaced))},
		{"```~ preformat ~```", "\n~ preformat ~\n", fmts(NewFormat(1, 14, Preformatted))},

		// Emoji
		{":emo_ji:", "emo_ji", fmts(NewFormat(0, 6, Emoji))},
		{":+1:", "+1", fmts(NewFormat(0, 2, Emoji))},
		{":not an emo ji:", ":not an emo ji:", make([]format, 0)},

		// Aggregate
		{"<@id|user> <!here|@here> *Check* _this_ `out`!:\n```&<>_```\n *in my* <#id|channel>.",
			"user @here Check this out!:\n&<>_\n in my #channel.",
			fmts(
				NewFormat(0, 4, User),
				NewFormat(5, 10, Variable),
				NewFormat(11, 16, Bold),
				NewFormat(17,21, Italic),
				NewFormat(22,25, Monospaced),
				NewFormat(28,32, Preformatted),
				NewFormat(34,39, Bold),
				NewFormat(40,48, _Channel))},
	}

	fe := Formatter{ lookup: &stubLookup{} }
	for _, data := range tests {
		content, formatting := fe.Format(data.s)
		if data.content != string(content) {
			t.Errorf(errorString, "Content", data.s, string(content), data.content)
		}
		if !reflect.DeepEqual(data.formatting, formatting) {
			t.Errorf(errorString, "Formatting", data.s, formatting, data.formatting)
		}
	}
}

func fmts(f ...format) []format {
	return f
}

// ---------------------------------------------------------------------------------------------------------------------
//
// Stub lookup for test
//
// ---------------------------------------------------------------------------------------------------------------------

type stubLookup struct {}

func (sl *stubLookup) GetUser(id string) string {
	return "user"
}

func (sl *stubLookup) GetChannel(id string) string {
	return "channel"
}