package template

import (
	_ "embed"
)

//go:embed model/agent-polygon-database-querier.md
var ModelAgentPolygonDatabaseQuerier []byte

//go:embed model/agent-polygon-payload-manager.md
var ModelAgentPolygonPayloadManager []byte

//go:embed model/agent-polygon-structural-handler-writer.md
var ModelAgentPolygonStructuralHandlerWriter []byte

//go:embed model/agent-polygon-structural-service-writer.md
var ModelAgentPolygonStructuralServiceWriter []byte

//go:embed model/claude.md
var ModelClaude []byte

//go:embed structure/makefile
var StructureMakefile []byte
