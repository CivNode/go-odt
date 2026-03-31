package odt

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"hash/crc32"
	"io"
	"strings"
)

// Render produces the complete ODT file as a byte slice.
func (d *Document) Render() ([]byte, error) {
	var buf bytes.Buffer
	if _, err := d.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteTo writes the complete ODT file to w.
func (d *Document) WriteTo(w io.Writer) (int64, error) {
	cw := &countWriter{w: w}
	zw := zip.NewWriter(cw)

	// 1. mimetype — MUST be first, uncompressed, no data descriptor,
	// no extra field. ODF 1.2, section 2.2.1.
	mimeContent := []byte("application/vnd.oasis.opendocument.text")
	mimeHeader := &zip.FileHeader{
		Name:               "mimetype",
		Method:             zip.Store,
		CRC32:              crc32.ChecksumIEEE(mimeContent),
		CompressedSize64:   uint64(len(mimeContent)),
		UncompressedSize64: uint64(len(mimeContent)),
	}
	mw, err := zw.CreateRaw(mimeHeader)
	if err != nil {
		return cw.n, fmt.Errorf("odt: create mimetype: %w", err)
	}
	if _, err := mw.Write(mimeContent); err != nil {
		return cw.n, fmt.Errorf("odt: write mimetype: %w", err)
	}

	// 2. Collect automatic styles from document content.
	autoStyles := d.collectAutoStyles()

	// 3. Write XML files.
	xmlFiles := []struct {
		name    string
		content string
	}{
		{"META-INF/manifest.xml", manifestXML},
		{"meta.xml", d.metaXML()},
		{"styles.xml", d.stylesXML()},
		{"content.xml", d.contentXML(autoStyles)},
	}
	for _, f := range xmlFiles {
		fw, err := zw.Create(f.name)
		if err != nil {
			return cw.n, fmt.Errorf("odt: create %s: %w", f.name, err)
		}
		if _, err := io.WriteString(fw, f.content); err != nil {
			return cw.n, fmt.Errorf("odt: write %s: %w", f.name, err)
		}
	}

	if err := zw.Close(); err != nil {
		return cw.n, fmt.Errorf("odt: close zip: %w", err)
	}
	return cw.n, nil
}

// countWriter wraps an io.Writer and counts bytes written.
type countWriter struct {
	w io.Writer
	n int64
}

func (c *countWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// autoStyle maps a styleKey string (e.g., "BI") to a generated name (e.g., "T1").
type autoStyle struct {
	name string
	span Span // representative span for property generation
}

func (d *Document) collectAutoStyles() map[string]autoStyle {
	seen := map[string]autoStyle{}
	counter := 0
	for _, b := range d.blocks {
		for _, spans := range d.spansInBlock(b) {
			for _, s := range spans {
				key := s.styleKey()
				if key == "" || !s.needsStyle() {
					continue
				}
				if _, ok := seen[key]; !ok {
					counter++
					seen[key] = autoStyle{
						name: fmt.Sprintf("T%d", counter),
						span: s,
					}
				}
			}
		}
	}
	return seen
}

func (d *Document) spansInBlock(b block) [][]Span {
	switch b.kind {
	case blockParagraph, blockBlockquote:
		return [][]Span{b.spans}
	case blockBulletList, blockOrderedList:
		return b.items
	case blockTable:
		all := make([][]Span, 0, len(b.headers)+len(b.rows))
		all = append(all, b.headers...)
		all = append(all, b.rows...)
		return all
	default:
		return nil
	}
}

// --- XML generation ---

const manifestXML = `<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">
  <manifest:file-entry manifest:media-type="application/vnd.oasis.opendocument.text" manifest:full-path="/"/>
  <manifest:file-entry manifest:media-type="text/xml" manifest:full-path="content.xml"/>
  <manifest:file-entry manifest:media-type="text/xml" manifest:full-path="styles.xml"/>
  <manifest:file-entry manifest:media-type="text/xml" manifest:full-path="meta.xml"/>
</manifest:manifest>`

func (d *Document) metaXML() string {
	title := d.title
	if title == "" {
		title = "Untitled"
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0"
  office:version="1.2">
  <office:meta>
    <dc:title>%s</dc:title>
    <dc:creator>%s</dc:creator>
    <dc:date>%s</dc:date>
    <meta:generator>%s</meta:generator>
  </office:meta>
</office:document-meta>`,
		esc(title), esc(d.author),
		d.created.Format("2006-01-02T15:04:05"),
		esc(d.generator))
}

func (d *Document) stylesXML() string {
	sz := d.fontSizePt
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
  xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  office:version="1.2">
  <office:styles>
    <style:default-style style:family="paragraph">
      <style:paragraph-properties fo:margin-top="0in" fo:margin-bottom="0.08in"
        fo:line-height="%[1]s"/>
      <style:text-properties fo:font-family="'%[2]s'" fo:font-size="%[3]dpt"/>
    </style:default-style>
    <style:style style:name="Standard" style:family="paragraph"/>
    <style:style style:name="Text_Body" style:family="paragraph" style:parent-style-name="Standard">
      <style:paragraph-properties fo:margin-top="0in" fo:margin-bottom="0.08in"/>
    </style:style>
    <style:style style:name="Heading_1" style:family="paragraph" style:parent-style-name="Standard"
      style:default-outline-level="1">
      <style:paragraph-properties fo:margin-top="0.3in" fo:margin-bottom="0.15in" fo:keep-with-next="always"/>
      <style:text-properties fo:font-size="%[4]dpt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_2" style:family="paragraph" style:parent-style-name="Standard"
      style:default-outline-level="2">
      <style:paragraph-properties fo:margin-top="0.25in" fo:margin-bottom="0.1in" fo:keep-with-next="always"/>
      <style:text-properties fo:font-size="%[5]dpt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_3" style:family="paragraph" style:parent-style-name="Standard"
      style:default-outline-level="3">
      <style:paragraph-properties fo:margin-top="0.2in" fo:margin-bottom="0.08in" fo:keep-with-next="always"/>
      <style:text-properties fo:font-size="%[6]dpt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_4" style:family="paragraph" style:parent-style-name="Standard"
      style:default-outline-level="4">
      <style:paragraph-properties fo:margin-top="0.15in" fo:margin-bottom="0.08in" fo:keep-with-next="always"/>
      <style:text-properties fo:font-size="%[7]dpt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_5" style:family="paragraph" style:parent-style-name="Standard"
      style:default-outline-level="5">
      <style:paragraph-properties fo:margin-top="0.1in" fo:margin-bottom="0.08in" fo:keep-with-next="always"/>
      <style:text-properties fo:font-size="%[3]dpt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_6" style:family="paragraph" style:parent-style-name="Standard"
      style:default-outline-level="6">
      <style:paragraph-properties fo:margin-top="0.1in" fo:margin-bottom="0.08in" fo:keep-with-next="always"/>
      <style:text-properties fo:font-size="%[3]dpt" fo:font-weight="bold" fo:font-style="italic"/>
    </style:style>
    <style:style style:name="Quotations" style:family="paragraph" style:parent-style-name="Standard">
      <style:paragraph-properties fo:margin-left="0.5in" fo:margin-right="0.5in"
        fo:margin-top="0.08in" fo:margin-bottom="0.08in"/>
      <style:text-properties fo:font-style="italic"/>
    </style:style>
    <style:style style:name="Preformatted_Text" style:family="paragraph" style:parent-style-name="Standard">
      <style:text-properties fo:font-family="'Courier New'" fo:font-size="10pt"/>
    </style:style>
    <style:style style:name="Separator" style:family="paragraph" style:parent-style-name="Standard">
      <style:paragraph-properties fo:text-align="center" fo:margin-top="0.2in" fo:margin-bottom="0.2in"/>
    </style:style>
    <style:style style:name="List_Paragraph" style:family="paragraph" style:parent-style-name="Standard">
      <style:paragraph-properties fo:margin-left="0.5in"/>
    </style:style>
    <style:style style:name="Table_Contents" style:family="paragraph" style:parent-style-name="Standard">
      <style:paragraph-properties fo:margin-top="0.02in" fo:margin-bottom="0.02in"/>
    </style:style>
    <style:style style:name="Table_Heading" style:family="paragraph" style:parent-style-name="Table_Contents">
      <style:text-properties fo:font-weight="bold"/>
    </style:style>
  </office:styles>
  <office:automatic-styles>
    <style:page-layout style:name="pm1">
      <style:page-layout-properties fo:page-width="8.5in" fo:page-height="11in"
        fo:margin-top="%[8]s" fo:margin-bottom="%[9]s"
        fo:margin-left="%[10]s" fo:margin-right="%[11]s"/>
    </style:page-layout>
  </office:automatic-styles>
  <office:master-styles>
    <style:master-page style:name="Standard" style:page-layout-name="pm1"/>
  </office:master-styles>
</office:document-styles>`,
		lineSpacingValue(d.lineSpacing), // 1
		esc(d.fontFamily),               // 2
		sz,                              // 3: body size
		sz+4,                            // 4: h1
		sz+2,                            // 5: h2
		sz+1,                            // 6: h3
		sz,                              // 7: h4
		d.marginTop,                     // 8
		d.marginBot,                     // 9
		d.marginLeft,                    // 10
		d.marginRight,                   // 11
	)
}

func lineSpacingValue(mult float64) string {
	pct := int(mult * 100)
	return fmt.Sprintf("%d%%", pct)
}

func (d *Document) contentXML(autoStyles map[string]autoStyle) string {
	var b strings.Builder

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  xmlns:table="urn:oasis:names:tc:opendocument:xmlns:table:1.0"
  xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0"
  xmlns:xlink="http://www.w3.org/1999/xlink"
  office:version="1.2">
`)

	// Automatic styles for combined text formatting.
	b.WriteString("<office:automatic-styles>\n")
	// Page-break paragraph style.
	b.WriteString(`  <style:style style:name="P_Break" style:family="paragraph" style:parent-style-name="Standard">
    <style:paragraph-properties fo:break-before="page"/>
  </style:style>
`)
	for _, as := range sortedAutoStyles(autoStyles) {
		fmt.Fprintf(&b, `  <style:style style:name="%s" style:family="text">
    <style:text-properties`, as.name)
		s := as.span
		if s.bold {
			b.WriteString(` fo:font-weight="bold"`)
		}
		if s.italic {
			b.WriteString(` fo:font-style="italic"`)
		}
		if s.underline {
			b.WriteString(` style:text-underline-style="solid" style:text-underline-width="auto"`)
		}
		if s.strikethrough {
			b.WriteString(` style:text-line-through-style="solid"`)
		}
		if s.code {
			b.WriteString(` fo:font-family="'Courier New'" fo:font-size="10pt"`)
		}
		b.WriteString("/>\n  </style:style>\n")
	}
	b.WriteString("</office:automatic-styles>\n")

	b.WriteString("<office:body>\n<office:text>\n")

	for _, blk := range d.blocks {
		d.renderBlock(&b, blk, autoStyles)
	}

	b.WriteString("</office:text>\n</office:body>\n</office:document-content>")
	return b.String()
}

func (d *Document) renderBlock(b *strings.Builder, blk block, autoStyles map[string]autoStyle) {
	switch blk.kind {
	case blockParagraph:
		b.WriteString(`<text:p text:style-name="Text_Body">`)
		writeSpans(b, blk.spans, autoStyles)
		b.WriteString("</text:p>\n")

	case blockHeading:
		fmt.Fprintf(b, `<text:h text:style-name="Heading_%d" text:outline-level="%d">`,
			blk.level, blk.level)
		b.WriteString(esc(blk.text))
		b.WriteString("</text:h>\n")

	case blockBlockquote:
		b.WriteString(`<text:p text:style-name="Quotations">`)
		writeSpans(b, blk.spans, autoStyles)
		b.WriteString("</text:p>\n")

	case blockCodeBlock:
		text := strings.TrimRight(blk.text, "\n")
		for _, line := range strings.Split(text, "\n") {
			fmt.Fprintf(b, `<text:p text:style-name="Preformatted_Text">%s</text:p>`+"\n", esc(line))
		}

	case blockBulletList, blockOrderedList:
		b.WriteString("<text:list>\n")
		for _, item := range blk.items {
			b.WriteString("<text:list-item>\n")
			b.WriteString(`<text:p text:style-name="List_Paragraph">`)
			writeSpans(b, item, autoStyles)
			b.WriteString("</text:p>\n")
			b.WriteString("</text:list-item>\n")
		}
		b.WriteString("</text:list>\n")

	case blockTable:
		b.WriteString("<table:table>\n")
		for i := 0; i < blk.numCols; i++ {
			b.WriteString("<table:table-column/>\n")
		}
		// Header row.
		if len(blk.headers) > 0 {
			b.WriteString("<table:table-header-rows>\n<table:table-row>\n")
			for _, cell := range blk.headers {
				b.WriteString("<table:table-cell>\n")
				b.WriteString(`<text:p text:style-name="Table_Heading">`)
				writeSpans(b, cell, autoStyles)
				b.WriteString("</text:p>\n")
				b.WriteString("</table:table-cell>\n")
			}
			b.WriteString("</table:table-row>\n</table:table-header-rows>\n")
		}
		// Data rows.
		for i := 0; i < len(blk.rows); i += blk.numCols {
			b.WriteString("<table:table-row>\n")
			end := i + blk.numCols
			if end > len(blk.rows) {
				end = len(blk.rows)
			}
			for _, cell := range blk.rows[i:end] {
				b.WriteString("<table:table-cell>\n")
				b.WriteString(`<text:p text:style-name="Table_Contents">`)
				writeSpans(b, cell, autoStyles)
				b.WriteString("</text:p>\n")
				b.WriteString("</table:table-cell>\n")
			}
			b.WriteString("</table:table-row>\n")
		}
		b.WriteString("</table:table>\n")

	case blockHRule:
		b.WriteString(`<text:p text:style-name="Separator">* * *</text:p>` + "\n")

	case blockPageBreak:
		b.WriteString(`<text:p text:style-name="P_Break"/>` + "\n")
	}
}

func writeSpans(b *strings.Builder, spans []Span, autoStyles map[string]autoStyle) {
	for _, s := range spans {
		if s.text == "" {
			continue
		}
		if s.href != "" {
			fmt.Fprintf(b, `<text:a xlink:type="simple" xlink:href="%s">%s</text:a>`,
				esc(s.href), esc(s.text))
			continue
		}
		if !s.needsStyle() {
			b.WriteString(esc(s.text))
			continue
		}
		key := s.styleKey()
		if as, ok := autoStyles[key]; ok {
			fmt.Fprintf(b, `<text:span text:style-name="%s">%s</text:span>`,
				as.name, esc(s.text))
		} else {
			b.WriteString(esc(s.text))
		}
	}
}

// sortedAutoStyles returns auto styles sorted by name for deterministic output.
func sortedAutoStyles(m map[string]autoStyle) []autoStyle {
	result := make([]autoStyle, 0, len(m))
	for _, as := range m {
		result = append(result, as)
	}
	// Sort by name (T1, T2, ...) — simple since names are generated in order.
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].name < result[i].name {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// esc escapes a string for safe inclusion in XML content.
func esc(s string) string {
	var b strings.Builder
	xml.Escape(&b, []byte(s))
	return b.String()
}
