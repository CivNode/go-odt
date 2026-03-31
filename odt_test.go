package odt

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	doc := New()
	if doc.fontFamily != "Times New Roman" {
		t.Errorf("default font = %q, want Times New Roman", doc.fontFamily)
	}
	if doc.fontSizePt != 12 {
		t.Errorf("default size = %d, want 12", doc.fontSizePt)
	}
	if doc.generator != "go-odt" {
		t.Errorf("default generator = %q, want go-odt", doc.generator)
	}
}

func TestMetadata(t *testing.T) {
	doc := New()
	doc.SetTitle("Test Title")
	doc.SetAuthor("Test Author")
	doc.SetGenerator("MyApp")

	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	meta := readZipEntry(t, data, "meta.xml")
	for _, want := range []string{"Test Title", "Test Author", "MyApp"} {
		if !strings.Contains(meta, want) {
			t.Errorf("meta.xml missing %q", want)
		}
	}
}

func TestMimetypeEntry(t *testing.T) {
	doc := New()
	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	if len(data) < 78 {
		t.Fatal("output too small")
	}

	// Verify local file header at offset 0.
	sig := binary.LittleEndian.Uint32(data[0:4])
	if sig != 0x04034b50 {
		t.Fatalf("bad signature: %#x", sig)
	}

	flags := binary.LittleEndian.Uint16(data[6:8])
	if flags&0x8 != 0 {
		t.Error("mimetype entry has data descriptor flag set")
	}

	method := binary.LittleEndian.Uint16(data[8:10])
	if method != 0 {
		t.Errorf("mimetype compression method = %d, want 0 (Store)", method)
	}

	crc := binary.LittleEndian.Uint32(data[14:18])
	if crc == 0 {
		t.Error("mimetype CRC32 is zero in local header")
	}

	csize := binary.LittleEndian.Uint32(data[18:22])
	if csize != 39 {
		t.Errorf("mimetype compressed size = %d, want 39", csize)
	}

	usize := binary.LittleEndian.Uint32(data[22:26])
	if usize != 39 {
		t.Errorf("mimetype uncompressed size = %d, want 39", usize)
	}

	exlen := binary.LittleEndian.Uint16(data[28:30])
	if exlen != 0 {
		t.Errorf("mimetype extra field length = %d, want 0", exlen)
	}

	// Content at offset 38.
	content := string(data[38:77])
	if content != "application/vnd.oasis.opendocument.text" {
		t.Errorf("mimetype content = %q", content)
	}

	// Next entry should follow immediately (PK\x03\x04), no data descriptor.
	nextSig := binary.LittleEndian.Uint32(data[77:81])
	if nextSig != 0x04034b50 {
		t.Errorf("after mimetype: %#x, want next local file header (0x04034b50)", nextSig)
	}
}

func TestZipStructure(t *testing.T) {
	doc := New()
	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	required := map[string]bool{
		"mimetype":              false,
		"META-INF/manifest.xml": false,
		"meta.xml":              false,
		"styles.xml":            false,
		"content.xml":           false,
	}
	for _, f := range r.File {
		if _, ok := required[f.Name]; ok {
			required[f.Name] = true
		}
	}
	for name, found := range required {
		if !found {
			t.Errorf("missing required zip entry: %s", name)
		}
	}

	// mimetype must be the first entry.
	if r.File[0].Name != "mimetype" {
		t.Errorf("first zip entry = %q, want mimetype", r.File[0].Name)
	}
}

func TestParagraph(t *testing.T) {
	doc := New()
	doc.AddParagraph("Hello, world!")
	content := renderContent(t, doc)

	if !strings.Contains(content, `<text:p text:style-name="Text_Body">Hello, world!</text:p>`) {
		t.Error("paragraph not found in content.xml")
	}
}

func TestHeading(t *testing.T) {
	doc := New()
	doc.AddHeading(2, "Chapter Two")
	content := renderContent(t, doc)

	if !strings.Contains(content, `text:outline-level="2"`) {
		t.Error("heading outline level 2 not found")
	}
	if !strings.Contains(content, "Chapter Two") {
		t.Error("heading text not found")
	}
}

func TestHeadingLevelClamping(t *testing.T) {
	doc := New()
	doc.AddHeading(0, "Zero")
	doc.AddHeading(99, "NinetyNine")
	content := renderContent(t, doc)

	if !strings.Contains(content, `text:outline-level="1"`) {
		t.Error("level 0 not clamped to 1")
	}
	if !strings.Contains(content, `text:outline-level="6"`) {
		t.Error("level 99 not clamped to 6")
	}
}

func TestInlineFormatting(t *testing.T) {
	tests := []struct {
		name string
		span Span
		want string // substring expected in the auto style
	}{
		{"bold", Bold("x"), `fo:font-weight="bold"`},
		{"italic", Italic("x"), `fo:font-style="italic"`},
		{"underline", Underline("x"), `style:text-underline-style="solid"`},
		{"strikethrough", Strikethrough("x"), `style:text-line-through-style="solid"`},
		{"code", Code("x"), `fo:font-family="'Courier New'"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := New()
			doc.AddParagraph(tt.span)
			content := renderContent(t, doc)

			if !strings.Contains(content, tt.want) {
				t.Errorf("content.xml missing %q for %s formatting", tt.want, tt.name)
			}
			if !strings.Contains(content, "<text:span") {
				t.Error("no text:span found")
			}
		})
	}
}

func TestCombinedFormatting(t *testing.T) {
	doc := New()
	doc.AddParagraph(Text("both").Bold().Italic())
	content := renderContent(t, doc)

	// Should have a single automatic style with both properties.
	if !strings.Contains(content, `fo:font-weight="bold"`) {
		t.Error("missing bold in combined style")
	}
	if !strings.Contains(content, `fo:font-style="italic"`) {
		t.Error("missing italic in combined style")
	}
}

func TestLink(t *testing.T) {
	doc := New()
	doc.AddParagraph(Link("CivNode", "https://civnode.com"))
	content := renderContent(t, doc)

	if !strings.Contains(content, `xlink:href="https://civnode.com"`) {
		t.Error("link href not found")
	}
	if !strings.Contains(content, ">CivNode</text:a>") {
		t.Error("link text not found")
	}
}

func TestBlockquote(t *testing.T) {
	doc := New()
	doc.AddBlockquote("A quoted passage.")
	content := renderContent(t, doc)

	if !strings.Contains(content, `text:style-name="Quotations"`) {
		t.Error("blockquote style not found")
	}
	if !strings.Contains(content, "A quoted passage.") {
		t.Error("blockquote text not found")
	}
}

func TestCodeBlock(t *testing.T) {
	doc := New()
	doc.AddCodeBlock("line1\nline2\nline3")
	content := renderContent(t, doc)

	if !strings.Contains(content, `text:style-name="Preformatted_Text"`) {
		t.Error("code block style not found")
	}
	if strings.Count(content, "Preformatted_Text") != 3 {
		t.Errorf("expected 3 code lines, got %d", strings.Count(content, "Preformatted_Text"))
	}
}

func TestBulletList(t *testing.T) {
	doc := New()
	doc.AddList("Alpha", "Beta", "Gamma")
	content := renderContent(t, doc)

	if !strings.Contains(content, "<text:list>") {
		t.Error("list element not found")
	}
	if strings.Count(content, "<text:list-item>") != 3 {
		t.Errorf("expected 3 list items")
	}
	if !strings.Contains(content, "Alpha") {
		t.Error("list item text not found")
	}
}

func TestOrderedList(t *testing.T) {
	doc := New()
	doc.AddOrderedList("First", "Second")
	content := renderContent(t, doc)

	if !strings.Contains(content, "<text:list>") {
		t.Error("ordered list element not found")
	}
	if strings.Count(content, "<text:list-item>") != 2 {
		t.Error("expected 2 list items")
	}
}

func TestTable(t *testing.T) {
	doc := New()
	doc.AddTable(
		[]string{"Name", "Age"},
		[][]string{{"Alice", "30"}, {"Bob", "25"}},
	)
	content := renderContent(t, doc)

	if !strings.Contains(content, "<table:table>") {
		t.Error("table element not found")
	}
	if strings.Count(content, "<table:table-column/>") != 2 {
		t.Error("expected 2 table columns")
	}
	if !strings.Contains(content, "<table:table-header-rows>") {
		t.Error("table header rows not found")
	}
	if !strings.Contains(content, "Alice") {
		t.Error("table cell content not found")
	}
	// 1 header row + 2 data rows = 3 rows.
	if strings.Count(content, "<table:table-row>") != 3 {
		t.Errorf("expected 3 table rows, got %d", strings.Count(content, "<table:table-row>"))
	}
}

func TestHorizontalRule(t *testing.T) {
	doc := New()
	doc.AddHorizontalRule()
	content := renderContent(t, doc)

	if !strings.Contains(content, `text:style-name="Separator"`) {
		t.Error("separator style not found")
	}
	if !strings.Contains(content, "* * *") {
		t.Error("separator text not found")
	}
}

func TestPageBreak(t *testing.T) {
	doc := New()
	doc.AddParagraph("Page 1")
	doc.AddPageBreak()
	doc.AddParagraph("Page 2")
	content := renderContent(t, doc)

	if !strings.Contains(content, `style:name="P_Break"`) {
		t.Error("page break style not found")
	}
	if !strings.Contains(content, `fo:break-before="page"`) {
		t.Error("break-before property not found")
	}
}

func TestXMLEscape(t *testing.T) {
	doc := New()
	doc.SetTitle("A & B < C > D")
	doc.AddParagraph("x < y & z > w")
	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	meta := readZipEntry(t, data, "meta.xml")
	if strings.Contains(meta, "A & B") {
		t.Error("unescaped & in meta.xml")
	}
	if !strings.Contains(meta, "A &amp; B &lt; C &gt; D") {
		t.Error("title not properly escaped in meta.xml")
	}

	content := readZipEntry(t, data, "content.xml")
	if !strings.Contains(content, "x &lt; y &amp; z &gt; w") {
		t.Error("paragraph text not properly escaped")
	}
}

func TestEmptyDocument(t *testing.T) {
	doc := New()
	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	// Should still be valid zip with all required entries.
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.File) < 5 {
		t.Errorf("expected at least 5 zip entries, got %d", len(r.File))
	}

	// content.xml should have office:text even with no content.
	content := readZipEntry(t, data, "content.xml")
	if !strings.Contains(content, "<office:text>") {
		t.Error("missing office:text in empty document")
	}
}

func TestLargeDocument(t *testing.T) {
	doc := New()
	for i := 0; i < 1000; i++ {
		doc.AddParagraph("This is paragraph number ", Bold("one thousand"), ".")
	}
	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 1000 {
		t.Error("output suspiciously small for 1000 paragraphs")
	}
}

func TestWriteTo(t *testing.T) {
	doc := New()
	doc.AddParagraph("test")

	rendered, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	n, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatal(err)
	}

	if n != int64(len(buf.Bytes())) {
		t.Errorf("WriteTo returned n=%d but wrote %d bytes", n, len(buf.Bytes()))
	}
	if !bytes.Equal(rendered, buf.Bytes()) {
		t.Error("Render and WriteTo produced different output")
	}
}

func TestSpanChaining(t *testing.T) {
	s := Text("test").Bold().Italic().Underline()
	if !s.bold || !s.italic || !s.underline {
		t.Error("chained formatting not applied")
	}
	if s.strikethrough || s.code {
		t.Error("unexpected formatting applied")
	}
	if s.text != "test" {
		t.Errorf("text = %q, want test", s.text)
	}
}

func TestUTF8(t *testing.T) {
	doc := New()
	doc.SetTitle("Ünïcödé")
	doc.AddParagraph("Chinese: \u4e16\u754c")
	doc.AddParagraph("Arabic: \u0645\u0631\u062d\u0628\u0627")
	doc.AddParagraph("Emoji: \U0001f680\U0001f30d")

	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	content := readZipEntry(t, data, "content.xml")
	if !strings.Contains(content, "\u4e16\u754c") {
		t.Error("Chinese characters not preserved")
	}
	if !strings.Contains(content, "\U0001f680") {
		t.Error("emoji not preserved")
	}

	meta := readZipEntry(t, data, "meta.xml")
	if !strings.Contains(meta, "Ünïcödé") {
		t.Error("Unicode title not preserved")
	}
}

func TestContentXMLIsValidXML(t *testing.T) {
	doc := New()
	doc.SetTitle("Validation Test")
	doc.AddHeading(1, "Chapter")
	doc.AddParagraph("Normal text, ", Bold("bold"), ", ", Italic("italic"), ".")
	doc.AddParagraph(Text("combined").Bold().Italic())
	doc.AddBlockquote("A quote")
	doc.AddCodeBlock("code()")
	doc.AddList("one", "two")
	doc.AddTable([]string{"A"}, [][]string{{"1"}})
	doc.AddHorizontalRule()

	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	content := readZipEntry(t, data, "content.xml")
	decoder := xml.NewDecoder(strings.NewReader(content))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("content.xml is not valid XML: %v", err)
		}
	}
}

func TestStylesXMLIsValidXML(t *testing.T) {
	doc := New()
	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	styles := readZipEntry(t, data, "styles.xml")
	decoder := xml.NewDecoder(strings.NewReader(styles))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("styles.xml is not valid XML: %v", err)
		}
	}
}

func TestMetaXMLIsValidXML(t *testing.T) {
	doc := New()
	doc.SetTitle("Test <Title> & \"Quotes\"")
	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	meta := readZipEntry(t, data, "meta.xml")
	decoder := xml.NewDecoder(strings.NewReader(meta))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("meta.xml is not valid XML: %v", err)
		}
	}
}

func TestManifestXMLIsValidXML(t *testing.T) {
	doc := New()
	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	manifest := readZipEntry(t, data, "META-INF/manifest.xml")
	decoder := xml.NewDecoder(strings.NewReader(manifest))
	for {
		_, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatalf("manifest.xml is not valid XML: %v", err)
		}
	}
}

func TestSetFont(t *testing.T) {
	doc := New()
	doc.SetFont("Georgia", 14)
	doc.AddParagraph("text")

	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	styles := readZipEntry(t, data, "styles.xml")
	if !strings.Contains(styles, "Georgia") {
		t.Error("custom font not found in styles.xml")
	}
	if !strings.Contains(styles, "14pt") {
		t.Error("custom font size not found in styles.xml")
	}
}

func TestSetMargins(t *testing.T) {
	doc := New()
	doc.SetMargins("0.5in", "0.75in", "1in", "1.5in")

	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	styles := readZipEntry(t, data, "styles.xml")
	if !strings.Contains(styles, `fo:margin-top="0.5in"`) {
		t.Error("custom top margin not found")
	}
	if !strings.Contains(styles, `fo:margin-right="1.5in"`) {
		t.Error("custom right margin not found")
	}
}

func TestLineSpacing(t *testing.T) {
	doc := New()
	doc.SetLineSpacing(2.0)

	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}

	styles := readZipEntry(t, data, "styles.xml")
	if !strings.Contains(styles, `fo:line-height="200%"`) {
		t.Error("double line spacing not found in styles.xml")
	}
}

func TestAutoStyleDeduplication(t *testing.T) {
	doc := New()
	// Use bold in multiple paragraphs — should produce only one auto style.
	doc.AddParagraph(Bold("one"))
	doc.AddParagraph(Bold("two"))
	doc.AddParagraph(Bold("three"))
	content := renderContent(t, doc)

	// Count auto style definitions. Only one should exist for bold.
	count := strings.Count(content, `style:name="T`)
	if count != 1 {
		t.Errorf("expected 1 auto style for bold, got %d", count)
	}
}

// --- helpers ---

func renderContent(t *testing.T, doc *Document) string {
	t.Helper()
	data, err := doc.Render()
	if err != nil {
		t.Fatal(err)
	}
	return readZipEntry(t, data, "content.xml")
}

func readZipEntry(t *testing.T, data []byte, name string) string {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			buf.ReadFrom(rc)
			rc.Close()
			return buf.String()
		}
	}
	t.Fatalf("zip entry %q not found", name)
	return ""
}
