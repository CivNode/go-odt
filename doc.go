// Package odt generates OpenDocument Text (.odt) files.
//
// ODT is the open standard document format (ISO/IEC 26300) used by
// LibreOffice, OpenOffice, Google Docs (import/export), and many other
// applications. This package produces ODF 1.2 compliant files that open
// cleanly in all major word processors.
//
// Zero dependencies — uses only the Go standard library.
//
// Quick start:
//
//	doc := odt.New()
//	doc.SetTitle("My Document")
//	doc.SetAuthor("Jane Doe")
//	doc.AddHeading(1, "Introduction")
//	doc.AddParagraph("Hello, world!")
//	data, err := doc.Render()
package odt
