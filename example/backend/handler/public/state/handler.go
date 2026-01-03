package stateEndpoint

import (
	"go.scnd.dev/open/polygon"
)

type Handler struct {
	layer polygon.Layer
}

func Handle(
	polygon polygon.Polygon,
) *Handler {
	return &Handler{
		layer: polygon.Layer("public", "handler"),
	}
}
