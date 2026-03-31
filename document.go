package odt

import "time"

// Document represents an ODT document under construction.
// Create one with [New], add content with the Add methods, then call
// [Document.Render] to produce the ODT bytes.
type Document struct {
	title     string
	author    string
	generator string
	created   time.Time

	fontFamily  string
	fontSizePt  int
	lineSpacing float64
	marginTop   string
	marginBot   string
	marginLeft  string
	marginRight string

	blocks []block
}

type blockKind int

const (
	blockParagraph blockKind = iota
	blockHeading
	blockBlockquote
	blockCodeBlock
	blockBulletList
	blockOrderedList
	blockTable
	blockHRule
	blockPageBreak
)

type block struct {
	kind    blockKind
	spans   []Span
	text    string
	level   int
	items   [][]Span
	headers [][]Span
	rows    [][]Span
	numCols int
}

// New creates a new empty ODT document with sensible defaults.
func New() *Document {
	return &Document{
		generator:   "go-odt",
		created:     time.Now(),
		fontFamily:  "Times New Roman",
		fontSizePt:  12,
		lineSpacing: 1.15,
		marginTop:   "1in",
		marginBot:   "1in",
		marginLeft:  "1.25in",
		marginRight: "1.25in",
	}
}

// SetTitle sets the document title (appears in file metadata).
func (d *Document) SetTitle(title string) { d.title = title }

// SetAuthor sets the document author (appears in file metadata).
func (d *Document) SetAuthor(author string) { d.author = author }

// SetGenerator overrides the generator string in metadata (defaults to "go-odt").
func (d *Document) SetGenerator(gen string) { d.generator = gen }

// SetFont sets the default font family and size in points.
func (d *Document) SetFont(family string, sizePt int) {
	d.fontFamily = family
	d.fontSizePt = sizePt
}

// SetLineSpacing sets line height as a multiplier (e.g., 1.5 for 150%).
func (d *Document) SetLineSpacing(spacing float64) { d.lineSpacing = spacing }

// SetMargins sets page margins. Values use CSS-style units (e.g., "1in", "2.54cm").
func (d *Document) SetMargins(top, bottom, left, right string) {
	d.marginTop = top
	d.marginBot = bottom
	d.marginLeft = left
	d.marginRight = right
}
