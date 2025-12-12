package pid

type limit struct {
	lower float64
	upper float64
}

func newLimit(lower, upper float64) limit {
	return limit{
		lower: lower,
		upper: upper,
	}
}

// apply ensures that the given value falls within a range.
// - If the value is less than `lower`, you get `lower`.
// - If the value is greater than `upper`, you get `upper`.
// - If the value is between `lower` and `upper`, you get the value itself.
func (l limit) apply(value float64) float64 {
	return min(max(l.lower, value), l.upper)
}
