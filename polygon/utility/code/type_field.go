package code

type Field struct {
	Name      *string     `json:"name"`
	Type      *string     `json:"type"`
	Tags      []*Tag      `json:"tags"`
	Annotates []*Annotate `json:"annotates"`
}

type Tag struct {
	Name  *string
	Value *string
}
