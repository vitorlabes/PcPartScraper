package domain

type Product struct {
	Title    string
	Brand    string
	Price    float64
	RawPrice string
	Page     int
	Category string
}

func (p Product) UniqueKey() string {
	return p.Title + "|" + p.Category
}
