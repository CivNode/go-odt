package odt

// Span represents a run of inline text with optional formatting.
// Create spans with the constructor functions ([Text], [Bold], [Italic], etc.)
// and combine formats with the chainable methods.
//
//	odt.Bold("important")                     // bold
//	odt.Text("key point").Bold().Italic()     // bold + italic
//	odt.Link("CivNode", "https://civnode.com") // hyperlink
type Span struct {
	text          string
	bold          bool
	italic        bool
	underline     bool
	strikethrough bool
	code          bool
	href          string
}

// Text creates a plain text span.
func Text(s string) Span { return Span{text: s} }

// Bold creates a bold text span.
func Bold(s string) Span { return Span{text: s, bold: true} }

// Italic creates an italic text span.
func Italic(s string) Span { return Span{text: s, italic: true} }

// Underline creates an underlined text span.
func Underline(s string) Span { return Span{text: s, underline: true} }

// Strikethrough creates a struck-through text span.
func Strikethrough(s string) Span { return Span{text: s, strikethrough: true} }

// Code creates a monospace code span.
func Code(s string) Span { return Span{text: s, code: true} }

// Link creates a hyperlink span.
func Link(text, href string) Span { return Span{text: text, href: href} }

// Bold returns a copy of the span with bold enabled.
func (s Span) Bold() Span { s.bold = true; return s }

// Italic returns a copy of the span with italic enabled.
func (s Span) Italic() Span { s.italic = true; return s }

// Underline returns a copy of the span with underline enabled.
func (s Span) Underline() Span { s.underline = true; return s }

// Strikethrough returns a copy of the span with strikethrough enabled.
func (s Span) Strikethrough() Span { s.strikethrough = true; return s }

// Code returns a copy of the span with monospace code formatting enabled.
func (s Span) Code() Span { s.code = true; return s }

// styleKey returns a string key identifying this span's formatting.
// Used internally to deduplicate automatic styles.
func (s Span) styleKey() string {
	if s.href != "" {
		return ""
	}
	var k string
	if s.bold {
		k += "B"
	}
	if s.italic {
		k += "I"
	}
	if s.underline {
		k += "U"
	}
	if s.strikethrough {
		k += "S"
	}
	if s.code {
		k += "C"
	}
	return k
}

// needsStyle returns true if this span requires a text style.
func (s Span) needsStyle() bool {
	return s.bold || s.italic || s.underline || s.strikethrough || s.code
}
