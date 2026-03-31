# go-odt

[![Go Reference](https://pkg.go.dev/badge/github.com/CivNode/go-odt.svg)](https://pkg.go.dev/github.com/CivNode/go-odt)
[![CI](https://github.com/CivNode/go-odt/actions/workflows/ci.yml/badge.svg)](https://github.com/CivNode/go-odt/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/CivNode/go-odt)](https://goreportcard.com/report/github.com/CivNode/go-odt)

Generate OpenDocument Text (.odt) files in Go. Zero dependencies.

## Why ODT?

ODT is the open document standard (ISO/IEC 26300). LibreOffice, OpenOffice, Google Docs, and Microsoft Word all open it. Unlike DOCX, the format is straightforward: a ZIP archive containing XML files. If you need to generate documents programmatically and don't want to pull in a massive library, this does the job.

## Install

```
go get github.com/CivNode/go-odt
```

Requires Go 1.22 or later. No external dependencies.

## Quick start

```go
package main

import (
    "os"
    "github.com/CivNode/go-odt"
)

func main() {
    doc := odt.New()
    doc.SetTitle("Quarterly Report")
    doc.SetAuthor("Engineering")

    doc.AddHeading(1, "Summary")
    doc.AddParagraph("Revenue grew ", odt.Bold("23%"), " quarter over quarter.")
    doc.AddParagraph("Details follow in the sections below.")

    doc.AddHeading(2, "Metrics")
    doc.AddTable(
        []string{"Metric", "Q1", "Q2"},
        [][]string{
            {"Users", "1,200", "1,850"},
            {"Revenue", "$45k", "$55k"},
        },
    )

    doc.AddHeading(2, "Next Steps")
    doc.AddList("Expand to EU markets", "Hire two more engineers", "Ship v2.0")

    data, err := doc.Render()
    if err != nil {
        panic(err)
    }
    os.WriteFile("report.odt", data, 0644)
}
```

## API

### Document

```go
doc := odt.New()                                    // new document with sensible defaults
doc.SetTitle("My Document")                         // metadata: title
doc.SetAuthor("Jane Doe")                           // metadata: author
doc.SetGenerator("MyApp v1.0")                      // metadata: generator (defaults to "go-odt")
doc.SetFont("Georgia", 14)                          // font family and size in points
doc.SetLineSpacing(1.5)                             // line height multiplier
doc.SetMargins("1in", "1in", "1.25in", "1.25in")   // top, bottom, left, right
```

### Block content

```go
doc.AddHeading(1, "Chapter Title")                  // headings, levels 1-6
doc.AddParagraph("Text with ", odt.Bold("bold"))    // paragraphs with inline formatting
doc.AddBlockquote("A quoted passage.")              // indented italic block
doc.AddCodeBlock("func main() {}")                  // monospace preformatted block
doc.AddList("First", "Second", "Third")             // bullet list
doc.AddOrderedList("Step 1", "Step 2")              // numbered list
doc.AddTable(headers, rows)                         // table with headers
doc.AddHorizontalRule()                             // section separator (****)
doc.AddPageBreak()                                  // page break
```

### Inline formatting

Constructor functions for single formats:

```go
odt.Bold("text")
odt.Italic("text")
odt.Underline("text")
odt.Strikethrough("text")
odt.Code("text")
odt.Link("display text", "https://example.com")
```

Chain methods to combine formats:

```go
odt.Text("important").Bold().Italic()               // bold + italic
odt.Text("deprecated").Strikethrough().Code()        // monospace strikethrough
```

`AddParagraph`, `AddBlockquote`, and list items all accept a mix of plain strings and `Span` values:

```go
doc.AddParagraph(
    "See ",
    odt.Link("the docs", "https://example.com"),
    " for ",
    odt.Bold("complete"),
    " details.",
)
```

### Render

```go
data, err := doc.Render()       // returns []byte
n, err := doc.WriteTo(writer)   // writes to any io.Writer
```

## ODF compliance

The output conforms to ODF 1.2 (the version LibreOffice expects). Specifically:

- **mimetype entry** is the first file in the ZIP, stored without compression, with no data descriptor and no extra field in the local header. This is the part most Go ZIP libraries get wrong (Go's `archive/zip` adds data descriptors by default). We use `CreateRaw` with pre-computed CRC32 to avoid it.
- **manifest.xml** lists all content files with correct media types.
- **styles.xml** defines named paragraph styles (headings, body, blockquote, code, etc.) and page layout with configurable margins.
- **content.xml** uses automatic styles for combined inline formatting. If you use bold+italic on the same text, it generates a proper combined `<style:style>` rather than picking one and dropping the other.
- All XML output is properly escaped and parses cleanly.

Files open without warnings in LibreOffice, OpenOffice, and Google Docs.

## What this library does not do

- Read or parse existing ODT files
- Embed images (images are rendered as alt-text placeholders)
- Generate spreadsheets (.ods) or presentations (.odp)
- Handle right-to-left text layout
- Produce PDF directly (use LibreOffice headless for that: `libreoffice --convert-to pdf doc.odt`)

These may come in future versions if there's demand.

## Used in production

[civnode.com](https://civnode.com) uses go-odt for book export. The library was extracted from CivNode's internal export pipeline and published as a standalone package.

## License

MIT
