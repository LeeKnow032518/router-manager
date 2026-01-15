// internal/repository/postgres/PostgresRepository_test.go
//go:build integration
// +build integration

package postgres_test

import (
	"context"
	"router-manager/internal/model"
	"router-manager/testhelper"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresRepository(t *testing.T) {
	testDb := testhelper.SetupTestPostgres(t)

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

	err := testDb.Repo.SaveRouter(context.Background(), router)
	assert.NoError(t, err)

	err = testDb.Repo.SaveCommand(context.Background(), command)
	assert.NoError(t, err)

	commandResult, err := testDb.Repo.GetCommandsByRouterId(context.Background(), routerId)
	require.NoError(t, err)
	assert.Equal(t, &commandResult[0], command)

	commandResult, _ = testDb.Repo.GetCommandsByRouterId(context.Background(), uuid.New())
	assert.Nil(t, commandResult)

	err = testDb.Repo.ChangeStatusByRouterId(context.Background(), routerId, "SENT")
	assert.NoError(t, err)

	routerResult, err := testDb.Repo.FindRouterByRouterId(context.Background(), routerId.String())
	assert.NoError(t, err)
	assert.Equal(t, routerResult.ID, routerId)
	assert.Equal(t, routerResult.SerialNumber, router.SerialNumber)
}
