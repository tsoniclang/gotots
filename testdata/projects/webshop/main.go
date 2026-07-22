// Command webshop demonstrates the shop.
package main

import (
	"fmt"

	"shop.example/webshop/catalog"
	"shop.example/webshop/order"
)

func main() {
	c := catalog.New()
	p := catalog.Product{SKU: "A1", Name: "Widget", Cents: 500}
	if err := c.Add(p); err != nil {
		panic(err)
	}
	lines := []order.Line{{Product: p, Count: 2}}
	total := order.Total(lines, catalog.FlatDiscount{Off: 100})
	fmt.Println(total, order.SKUs(lines))
}
