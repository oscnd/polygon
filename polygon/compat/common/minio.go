package common

import (
	"net/url"

	"github.com/bsthun/gut"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioConfig interface {
	GetMinioEndpoint() *string
	GetMinioAccessKey() *string
	GetMinioSecretKey() *string
}

func Minio(config MinioConfig) *minio.Client {
	// * initialize minio client
	parsed, err := url.Parse(*config.GetMinioEndpoint())
	if err != nil {
		gut.Fatal("failed to parse minio endpoint", err)
	}

	minioClient, err := minio.New(parsed.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(*config.GetMinioAccessKey(), *config.GetMinioSecretKey(), ""),
		Secure: parsed.Scheme == "https",
	})

	if err != nil {
		gut.Fatal("failed to initialize minio", err)
	}

	return minioClient
}
