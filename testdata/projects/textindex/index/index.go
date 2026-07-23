// Package index maintains a concurrent document index.
package index

import (
	"sync"

	"index.example/textindex/tokenize"
)

// Doc is one indexed document.
type Doc struct {
	ID   int
	Text string
}

// Index maps words to the documents containing them.
type Index struct {
	mu    sync.Mutex
	byWord map[string][]int
}

// New builds an empty index.
func New() *Index {
	return &Index{byWord: map[string][]int{}}
}

// AddAll indexes documents from a channel until it closes.
func (ix *Index) AddAll(docs <-chan Doc) {
	for doc := range docs {
		for word := range tokenize.Counts(doc.Text) {
			ix.mu.Lock()
			ix.byWord[word] = append(ix.byWord[word], doc.ID)
			ix.mu.Unlock()
		}
	}
}

// Find returns the documents containing word.
func (ix *Index) Find(word string) []int {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	return ix.byWord[word]
}
