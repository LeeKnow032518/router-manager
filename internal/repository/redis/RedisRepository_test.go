// internal/repository/postgres/PostgresRepository_test.go
//go:build integration
// +build integration

package redis_test

import (
	"context"
	"router-manager/internal/model"
	"router-manager/internal/repository/redis"
	"router-manager/testhelper"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRedisRepository(t *testing.T) {
	testRedis := testhelper.SetupTestRedis(t)

	repo := redis.NewRedisRepository(testRedis.Client)

	routerId := uuid.New()
	commandId := uuid.New()

	router := &model.Router{
		ID:           routerId,
		SerialNumber: "SN123",
		CreatedAt:    time.Now(),
	}

	command := &model.Command{
		ID:          commandId,
		RouterID:    routerId,
		CommandType: "REBOOT",
		Status:      "PENDING",
	}

	err := repo.SaveCommand(context.Background(), command)
	assert.NoError(t, err)

	err = repo.SaveRouter(context.Background(), router)
	assert.NoError(t, err)

	resultCommand, err := repo.FindCommandsByRouterId(context.Background(), routerId)
	assert.NoError(t, err)
	assert.Equal(t, resultCommand[0].ID, commandId)
	assert.Equal(t, resultCommand[0].Status, command.Status)
	assert.Equal(t, resultCommand[0].RouterID, routerId)
	assert.Equal(t, resultCommand[0].CommandType, command.CommandType)

	err = repo.ChangeStatusByRouterId(context.Background(), routerId, "SENT")
	assert.NoError(t, err)

	err = repo.ChangeStatusByRouterId(context.Background(), routerId, "ACKED")
	assert.NoError(t, err)

	err = repo.ChangeStatusByRouterId(context.Background(), routerId, "SENT")
	assert.Error(t, err)

	resultRouter, err := repo.FindRouterByRouterId(context.Background(), routerId.String())
	assert.NoError(t, err)
	assert.NotNil(t, resultRouter)
	assert.Equal(t, resultRouter.ID, routerId)
	assert.Equal(t, resultRouter.SerialNumber, router.SerialNumber)
}
