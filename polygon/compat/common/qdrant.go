package common

import (
	"strconv"
	"strings"

	"github.com/bsthun/gut"
	"github.com/qdrant/go-client/qdrant"
)

type QdrantConfig interface {
	GetQdrantDsn() *string
	GetQdrantApiKey() *string
}

func Init(config QdrantConfig) *qdrant.Client {
	segments := strings.Split(*config.GetQdrantDsn(), ":")
	host := segments[0]
	port, err := strconv.ParseInt(segments[1], 10, 64)

	// * create qdrant client
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:                   host,
		Port:                   int(port),
		APIKey:                 *config.GetQdrantApiKey(),
		UseTLS:                 false,
		TLSConfig:              nil,
		GrpcOptions:            nil,
		SkipCompatibilityCheck: false,
	})
	if err != nil {
		gut.Fatal("unable to create qdrant client", err)
	}

	return client
}
