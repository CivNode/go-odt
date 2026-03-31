package odt_test

import (
	"fmt"
	"os"

	"github.com/CivNode/go-odt"
)

func ExampleNew() {
	doc := odt.New()
	doc.SetTitle("My Document")
	doc.SetAuthor("Jane Doe")
	doc.AddHeading(1, "Introduction")
	doc.AddParagraph("Hello, world!")
	data, err := doc.Render()
	if err != nil {
		panic(err)
	}
	fmt.Println(len(data) > 0)
	// Output: true
}

func ExampleDocument_AddParagraph() {
	doc := odt.New()
	doc.AddParagraph("Plain text, ", odt.Bold("bold"), ", and ", odt.Text("bold italic").Bold().Italic(), ".")
	doc.Render()
}

func ExampleDocument_WriteTo() {
	doc := odt.New()
	doc.AddParagraph("Written to a file.")

	f, err := os.CreateTemp("", "example-*.odt")
	if err != nil {
		panic(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	n, err := doc.WriteTo(f)
	if err != nil {
		panic(err)
	}
	fmt.Println(n > 0)
	// Output: true
}

func ExampleLink() {
	doc := odt.New()
	doc.AddParagraph("Visit ", odt.Link("CivNode", "https://civnode.com"), " for writing.")
	doc.Render()
}
