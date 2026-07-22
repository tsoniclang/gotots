// Package order totals purchases.
package order

import (
	"sort"

	"shop.example/webshop/catalog"
)

// Line is one purchased product count.
type Line struct {
	Product catalog.Product
	Count   int
}

// Total prices all lines through the given pricer.
func Total(lines []Line, pricer catalog.Pricer) int64 {
	var total int64
	for _, line := range lines {
		total += pricer.Price(line.Product) * int64(line.Count)
	}
	return total
}

// SKUs lists the purchased SKUs sorted.
func SKUs(lines []Line) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, line.Product.SKU)
	}
	sort.Strings(out)
	return out
}
