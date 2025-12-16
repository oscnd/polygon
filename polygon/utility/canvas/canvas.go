package canvas

type Canvas struct {
	Import *Import `json:"import,omitempty"`
}

func New() *Canvas {
	return &Canvas{
		Import: new(Import),
	}
}
