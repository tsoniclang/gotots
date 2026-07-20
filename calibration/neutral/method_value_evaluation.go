// Neutral semantic control: a method value evaluates its receiver
// exactly once, at creation time.
package neutral

type counter struct{ n int }

func (c *counter) Value() int { return c.n }

func Capture() (int, int) {
	c := &counter{n: 1}
	value := c.Value // receiver captured NOW
	c = &counter{n: 100}
	_ = c
	first := value()
	return first, first // both read the original receiver
}
