// Command textindex demonstrates the index.
package main

import (
	"fmt"

	"index.example/textindex/index"
)

func main() {
	ix := index.New()
	docs := make(chan index.Doc, 2)
	docs <- index.Doc{ID: 1, Text: "hello indexed world"}
	docs <- index.Doc{ID: 2, Text: "hello again"}
	close(docs)
	ix.AddAll(docs)
	fmt.Println(ix.Find("hello"))
}
