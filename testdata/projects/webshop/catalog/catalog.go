// Package catalog holds the product catalog.
package catalog

import "fmt"

// Product is one sellable item.
type Product struct {
	SKU   string
	Name  string
	Cents int64
}

// Pricer prices a product.
type Pricer interface {
	Price(p Product) int64
}

// FlatDiscount discounts every product by a fixed amount.
type FlatDiscount struct {
	Off int64
}

// Price implements Pricer.
func (d FlatDiscount) Price(p Product) int64 {
	if p.Cents < d.Off {
		return 0
	}
	return p.Cents - d.Off
}

// Catalog is a keyed product set.
type Catalog struct {
	products map[string]Product
}

// New builds an empty catalog.
func New() *Catalog {
	return &Catalog{products: map[string]Product{}}
}

// Add registers a product; duplicate SKUs are an error.
func (c *Catalog) Add(p Product) error {
	if _, exists := c.products[p.SKU]; exists {
		return fmt.Errorf("duplicate sku %s", p.SKU)
	}
	c.products[p.SKU] = p
	return nil
}

// Lookup finds a product by SKU.
func (c *Catalog) Lookup(sku string) (Product, bool) {
	p, ok := c.products[sku]
	return p, ok
}

// Map applies f to every value of a keyed set — a generic helper.
func Map[K comparable, V, W any](in map[K]V, f func(V) W) map[K]W {
	out := make(map[K]W, len(in))
	for k, v := range in {
		out[k] = f(v)
	}
	return out
}
