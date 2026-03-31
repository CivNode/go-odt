package odt

// toSpans converts a mixed list of string and Span values into []Span.
func toSpans(content []any) []Span {
	var spans []Span
	for _, c := range content {
		switch v := c.(type) {
		case string:
			spans = append(spans, Text(v))
		case Span:
			spans = append(spans, v)
		}
	}
	return spans
}

// AddParagraph adds a paragraph. Content arguments can be plain strings
// or [Span] values for inline formatting.
//
//	doc.AddParagraph("Hello, ", odt.Bold("world"), "!")
func (d *Document) AddParagraph(content ...any) {
	d.blocks = append(d.blocks, block{
		kind:  blockParagraph,
		spans: toSpans(content),
	})
}

// AddHeading adds a heading at the given outline level (1–6).
func (d *Document) AddHeading(level int, text string) {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	d.blocks = append(d.blocks, block{
		kind:  blockHeading,
		text:  text,
		level: level,
	})
}

// AddBlockquote adds a block quotation. Content arguments can be plain
// strings or [Span] values.
func (d *Document) AddBlockquote(content ...any) {
	d.blocks = append(d.blocks, block{
		kind:  blockBlockquote,
		spans: toSpans(content),
	})
}

// AddCodeBlock adds a preformatted code block.
func (d *Document) AddCodeBlock(text string) {
	d.blocks = append(d.blocks, block{
		kind: blockCodeBlock,
		text: text,
	})
}

// AddHorizontalRule adds a centered separator (e.g., "* * *").
func (d *Document) AddHorizontalRule() {
	d.blocks = append(d.blocks, block{kind: blockHRule})
}

// AddPageBreak inserts a page break before the next content.
func (d *Document) AddPageBreak() {
	d.blocks = append(d.blocks, block{kind: blockPageBreak})
}

// AddList adds a bullet list. Each item can be a string, a [Span],
// or a []any of mixed strings and Spans.
func (d *Document) AddList(items ...any) {
	d.blocks = append(d.blocks, block{
		kind:  blockBulletList,
		items: toListItems(items),
	})
}

// AddOrderedList adds a numbered list. Arguments are the same as [Document.AddList].
func (d *Document) AddOrderedList(items ...any) {
	d.blocks = append(d.blocks, block{
		kind:  blockOrderedList,
		items: toListItems(items),
	})
}

func toListItems(items []any) [][]Span {
	result := make([][]Span, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case string:
			result = append(result, []Span{Text(v)})
		case Span:
			result = append(result, []Span{v})
		case []any:
			result = append(result, toSpans(v))
		}
	}
	return result
}

// AddTable adds a table with column headers and data rows.
func (d *Document) AddTable(headers []string, rows [][]string) {
	numCols := len(headers)
	hdrSpans := make([][]Span, numCols)
	for i, h := range headers {
		hdrSpans[i] = []Span{Text(h)}
	}
	var rowSpans [][]Span
	for _, row := range rows {
		for _, cell := range row {
			rowSpans = append(rowSpans, []Span{Text(cell)})
		}
	}
	d.blocks = append(d.blocks, block{
		kind:    blockTable,
		headers: hdrSpans,
		rows:    rowSpans,
		numCols: numCols,
	})
}
