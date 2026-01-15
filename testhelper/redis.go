package testhelper

import (
	"context"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
)

type TestRedis struct {
	Client    *redis.Client
	Container testcontainers.Container
}

func SetupTestRedis(t *testing.T) *TestRedis {
	ctx := context.Background()

	redisContainer, err := rediscontainer.Run(ctx, "redis:latest")
	if err != nil {
		t.Fatalf("Failed to start Redis container: %v", err)
	}

	uri, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("Failed to get Redis connection string: %v", err)
	}

	addr := strings.TrimPrefix(uri, "redis://")

	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})

	return &TestRedis{
		Client:    client,
		Container: redisContainer,
	}
}
