package code

type Field struct {
	Name *string `json:"name"`
	Type *string `json:"type"`
	Tags []*Tag  `json:"tags"`
}

type Tag struct {
	Name  *string
	Value *string
}
