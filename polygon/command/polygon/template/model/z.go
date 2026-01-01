package model

import (
	_ "embed"
)

//go:embed agent-polygon-database-querier.md
var AgentPolygonDatabaseQuerier []byte

//go:embed agent-polygon-payload-manager.md
var AgentPolygonPayloadManager []byte

//go:embed agent-polygon-structural-handler-writer.md
var AgentPolygonStructuralHandlerWriter []byte

//go:embed agent-polygon-structural-service-writer.md
var AgentPolygonStructuralServiceWriter []byte

//go:embed claude.md
var Claude []byte
