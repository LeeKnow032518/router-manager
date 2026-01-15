package service

import (
	"context"
	"fmt"
	"router-manager/internal/model"
	"router-manager/internal/pb"
	mockspg "router-manager/internal/repository/postgres/mocks"
	mocksred "router-manager/internal/repository/redis/mocks"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (*CommandService, *mockspg.MockPostgresRepo, *mocksred.MockRedisRepo, context.Context) {
	ctx := context.Background()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// mock repositories
	mockPostgres := mockspg.NewMockPostgresRepo(ctrl)
	mockRedis := mocksred.NewMockRedisRepo(ctrl)

	s := NewCommandService(mockPostgres, mockRedis)

	return s, mockPostgres, mockRedis, ctx
}

/* --- test SendCommand method --- */

func TestSendCommand_ToManyRouters(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)

	req := &pb.SendCommandRequest{
		Routers: []*pb.Router{
			{RouterId: "a1b2c3d4-5678-90ef-1234-567890abcdef",
				SerialNumber: "SN123"},
			{RouterId: "a1b2c3d4-5678-90ef-1234-567890abcdet",
				SerialNumber: "SN124"},
		},
		CommandType: "REBOOT",
	}

	mockPostgres.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(2)

	mockRedis.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(2)

	mockPostgres.EXPECT().
		SaveCommand(gomock.Any(), gomock.AssignableToTypeOf(&model.Command{})).
		Return(nil).
		Times(2)

	mockRedis.EXPECT().
		SaveCommand(gomock.Any(), gomock.AssignableToTypeOf(&model.Command{})).
		Return(nil).
		Times(2)

	response, err := s.SendCommand(ctx, req)

	require.NotNil(t, response)
	require.NoError(t, err)
	assert.Equal(t, response.Status, "PENDING")
	assert.Len(t, response.Id, 2)
	assert.NotEmpty(t, response.Id[0])
	assert.NotEmpty(t, response.Id[1])
}

func TestSendCommand_toOneRouter(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)
	// expected results
	req := &pb.SendCommandRequest{
		Routers: []*pb.Router{
			{RouterId: "a1b2c3d4-5678-90ef-1234-567890abcdef",
				SerialNumber: "SN123"},
		},
		CommandType: "REBOOT",
	}

	mockPostgres.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockPostgres.EXPECT().
		SaveCommand(gomock.Any(), gomock.AssignableToTypeOf(&model.Command{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		SaveCommand(gomock.Any(), gomock.AssignableToTypeOf(&model.Command{})).
		Return(nil).
		Times(1)

	response, err := s.SendCommand(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "PENDING", response.Status)
	assert.Len(t, response.Id, 1)
	assert.NotEmpty(t, response.Id[0])
}

// test epty routers SendCommand
func TestSendCommand_EmptyRouters(t *testing.T) {
	s, _, _, ctx := setup(t)

	req := &pb.SendCommandRequest{
		Routers:     []*pb.Router{},
		CommandType: "REBOOT",
	}

	response, err := s.SendCommand(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no routers specified")
}

func TestSendCommand_EmptyCommandType(t *testing.T) {
	s, _, _, ctx := setup(t)

	req := &pb.SendCommandRequest{
		Routers: []*pb.Router{
			{RouterId: "a1b2c3d4-5678-90ef-1234-567890abcdef",
				SerialNumber: "SN123"},
		},
		CommandType: "",
	}

	response, err := s.SendCommand(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no command specified")
}

/* --- test PollCommands method --- */

func TestPollCommands(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)
	expectedUuid := uuid.New()

	expectedRouter := &model.Router{
		ID:           expectedUuid,
		SerialNumber: "SN123",
		IPAddress:    nil,
	}

	expectedCommands := []model.Command{
		{RouterID: expectedRouter.ID,
			CommandType: "REBOOT",
			Payload:     nil,
		},
	}

	mockRedis.EXPECT().
		FindRouterByRouterId(gomock.Any(), gomock.Eq(expectedUuid.String())).
		Return(expectedRouter, nil).
		Times(1)

	mockPostgres.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		FindCommandsByRouterId(gomock.Any(), expectedRouter.ID).
		Return(expectedCommands, nil).
		Times(1)

	mockRedis.EXPECT().
		ChangeStatusByRouterId(gomock.Any(), gomock.Eq(expectedUuid), gomock.Eq("SENT")).
		Return(nil).
		Times(1)

	mockPostgres.EXPECT().
		ChangeStatusByRouterId(gomock.Any(), gomock.Eq(expectedUuid), gomock.Eq("SENT")).
		Return(nil).
		Times(1)

	req := &pb.PollRequest{
		RouterId:     expectedUuid.String(),
		SerialNumber: "SN123",
	}

	response, err := s.PollCommands(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Len(t, response.Commands, 1)
	assert.NotEmpty(t, response.Commands[0])
	assert.Equal(t, response.Commands[0].CommandType, "REBOOT")
}

func TestPollCommands_EmptySerialNumber(t *testing.T) {
	s, _, _, ctx := setup(t)

	req := &pb.PollRequest{
		SerialNumber: "",
		RouterId:     uuid.NewString(),
	}

	response, err := s.PollCommands(ctx, req)

	require.Error(t, err)
	require.Nil(t, response)
	assert.Contains(t, err.Error(), "router serial_number is required")
}

func TestPollCommands_NoRouterFound(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)

	routerId := uuid.New().String()
	req := &pb.PollRequest{
		SerialNumber: "SN124",
		RouterId:     routerId,
	}

	mockPostgres.EXPECT().
		FindRouterByRouterId(ctx, gomock.Eq(routerId)).
		Return(nil, fmt.Errorf("failed to query database"))

	mockRedis.EXPECT().
		FindRouterByRouterId(ctx, gomock.Eq(routerId)).
		Return(nil, fmt.Errorf("failed to query database"))

	response, err := s.PollCommands(ctx, req)

	require.Error(t, err)
	require.Nil(t, response)
	assert.Contains(t, err.Error(), "there is no router with such serial_number")
}

func TestPollCommands_ErrorStatusChange(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)
	expectedUuid := uuid.New()

	expectedRouter := &model.Router{
		ID:           expectedUuid,
		SerialNumber: "SN123",
		IPAddress:    nil,
	}

	expectedCommands := []model.Command{
		{RouterID: expectedRouter.ID,
			CommandType: "REBOOT",
			Payload:     nil,
		},
	}

	mockRedis.EXPECT().
		FindRouterByRouterId(gomock.Any(), gomock.Eq(expectedUuid.String())).
		Return(expectedRouter, nil).
		Times(1)

	mockPostgres.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		FindCommandsByRouterId(gomock.Any(), expectedRouter.ID).
		Return(expectedCommands, nil).
		Times(1)

	mockRedis.EXPECT().
		ChangeStatusByRouterId(gomock.Any(), expectedUuid, gomock.Eq("SENT")).
		Return(fmt.Errorf("failed to change status")).
		Times(1)

	mockPostgres.EXPECT().
		ChangeStatusByRouterId(gomock.Any(), expectedUuid, gomock.Eq("SENT")).
		Return(fmt.Errorf("failed to change status")).
		Times(1)

	req := &pb.PollRequest{
		RouterId:     expectedUuid.String(),
		SerialNumber: "SN123",
	}

	response, err := s.PollCommands(ctx, req)

	require.Error(t, err)
	require.Nil(t, response)
}

/* --- test AckCommands method --- */

func TestAckCommand(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)

	expectedUuid := uuid.New()

	expectedRouter := &model.Router{
		ID:           expectedUuid,
		SerialNumber: "SN123",
		IPAddress:    nil,
	}

	mockRedis.EXPECT().
		FindRouterByRouterId(gomock.Any(), gomock.Eq(expectedUuid.String())).
		Return(expectedRouter, nil).
		Times(1)

	mockPostgres.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		ChangeStatusByRouterId(gomock.Any(), gomock.Eq(expectedUuid), gomock.Eq("ACKED")).
		Return(nil).
		Times(1)

	mockPostgres.EXPECT().
		ChangeStatusByRouterId(gomock.Any(), gomock.Eq(expectedUuid), gomock.Eq("ACKED")).
		Return(nil).
		Times(1)

	req := &pb.AckRequest{
		RouterId:     expectedUuid.String(),
		SerialNumber: expectedRouter.SerialNumber,
		CommandType:  "REBOOT",
	}

	response, err := s.AckCommand(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, response.Status, "ACKED")
}

func TestAckCommand_EmptySerialNumber(t *testing.T) {
	s, _, _, ctx := setup(t)

	req := &pb.AckRequest{
		RouterId:     uuid.NewString(),
		SerialNumber: "",
		CommandType:  "REBOOT",
	}

	response, err := s.AckCommand(ctx, req)

	require.Error(t, err)
	require.Nil(t, response)
	assert.Contains(t, err.Error(), "serial_number is required")
}

func TestAckCommand_EmptyCommandType(t *testing.T) {
	s, _, _, ctx := setup(t)

	req := &pb.AckRequest{
		RouterId:     uuid.NewString(),
		SerialNumber: "SN123",
		CommandType:  "",
	}

	response, err := s.AckCommand(ctx, req)

	require.Error(t, err)
	require.Nil(t, response)
	assert.Contains(t, err.Error(), "command_type is required")
}

func TestAckCommand_NoRouterFound(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)

	routerId := uuid.New().String()
	req := &pb.AckRequest{
		SerialNumber: "SN124",
		RouterId:     routerId,
		CommandType:  "REBOOT",
	}

	mockPostgres.EXPECT().
		FindRouterByRouterId(ctx, gomock.Eq(routerId)).
		Return(nil, fmt.Errorf("failed to query database"))

	mockRedis.EXPECT().
		FindRouterByRouterId(ctx, gomock.Eq(routerId)).
		Return(nil, fmt.Errorf("failed to query database"))

	response, err := s.AckCommand(ctx, req)

	require.Error(t, err)
	require.Nil(t, response)
	assert.Contains(t, err.Error(), "there is no router with such serial_number")
}

func TestAckCommands_ErrorStatusChange(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)
	expectedUuid := uuid.New()

	expectedRouter := &model.Router{
		ID:           expectedUuid,
		SerialNumber: "SN123",
		IPAddress:    nil,
	}

	expectedCommands := []model.Command{
		{RouterID: expectedRouter.ID,
			CommandType: "REBOOT",
			Payload:     nil,
		},
	}

	mockRedis.EXPECT().
		FindRouterByRouterId(gomock.Any(), gomock.Eq(expectedUuid.String())).
		Return(expectedRouter, nil).
		Times(1)

	mockPostgres.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		SaveRouter(gomock.Any(), gomock.AssignableToTypeOf(&model.Router{})).
		Return(nil).
		Times(1)

	mockRedis.EXPECT().
		FindCommandsByRouterId(gomock.Any(), expectedRouter.ID).
		Return(expectedCommands, nil).
		Times(1)

	mockRedis.EXPECT().
		ChangeStatusByRouterId(gomock.Any(), expectedUuid, gomock.Eq("ACKED")).
		Return(fmt.Errorf("failed to change status")).
		Times(1)

	mockPostgres.EXPECT().
		ChangeStatusByRouterId(gomock.Any(), expectedUuid, gomock.Eq("ACKED")).
		Return(fmt.Errorf("failed to change status")).
		Times(1)

	req := &pb.AckRequest{
		RouterId:     expectedUuid.String(),
		SerialNumber: "SN123",
		CommandType:  "REBOOT",
	}

	response, err := s.AckCommand(ctx, req)

	require.Error(t, err)
	require.Nil(t, response)
}

/* --- test findRouter method --- */

func TestFindRouter(t *testing.T) {
	s, _, mockRedis, ctx := setup(t)

	expectedUuid := uuid.New()

	expectedRouter := &model.Router{
		ID:           expectedUuid,
		SerialNumber: "SN123",
		IPAddress:    nil,
	}

	mockRedis.EXPECT().
		FindRouterByRouterId(ctx, gomock.Eq("SN123")).
		Return(expectedRouter, nil).
		Times(1)

	response := s.findRouter(ctx, "SN123")

	assert.Equal(t, response.SerialNumber, "SN123")
	assert.Equal(t, response.ID, expectedUuid)
}

func TestFindRouter_InPostgres(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)

	expectedUuid := uuid.New()

	expectedRouter := &model.Router{
		ID:           expectedUuid,
		SerialNumber: "SN123",
		IPAddress:    nil,
	}

	mockRedis.EXPECT().
		FindRouterByRouterId(ctx, gomock.Eq(expectedUuid.String())).
		Return(nil, fmt.Errorf("no such router")).
		Times(1)

	mockPostgres.EXPECT().
		FindRouterByRouterId(ctx, gomock.Eq(expectedUuid.String())).
		Return(expectedRouter, nil).
		Times(1)

	response := s.findRouter(ctx, expectedUuid.String())

	assert.Equal(t, response.ID, expectedUuid)
	assert.Equal(t, expectedRouter.SerialNumber, "SN123")
}

func TestFindRouter_NoRouter_Found(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)

	mockRedis.EXPECT().
		FindRouterByRouterId(ctx, gomock.Eq("SN123")).
		Return(nil, fmt.Errorf("no such router")).
		Times(1)

	mockPostgres.EXPECT().
		FindRouterByRouterId(ctx, gomock.Eq("SN123")).
		Return(nil, fmt.Errorf("no such router")).
		Times(1)

	response := s.findRouter(ctx, "SN123")

	assert.Nil(t, response)
}

/* --- test saveRouter method --- */

func TestSaveRouter(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)
	expectedUuid := uuid.New()

	expectedRouter := &model.Router{
		ID:           expectedUuid,
		SerialNumber: "SN123",
		IPAddress:    nil,
	}

	mockPostgres.EXPECT().
		SaveRouter(ctx, expectedRouter).
		Return(nil).Times(1)

	mockRedis.EXPECT().
		SaveRouter(ctx, expectedRouter).
		Return(nil).Times(1)

	s.SaveRouter(ctx, expectedRouter)
}

/* --- test ChangeStatus method --- */

func TestChangeStatus(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)
	expectedUuid := uuid.New()

	mockRedis.EXPECT().
		ChangeStatusByRouterId(ctx, expectedUuid, "SENT").
		Return(nil).Times(1)

	mockPostgres.EXPECT().
		ChangeStatusByRouterId(ctx, expectedUuid, "SENT").
		Return(nil).Times(1)

	err := s.ChangeStatus(ctx, expectedUuid, "SENT")

	assert.NoError(t, err)
}

func TestChangeStatus_ErrorInRedis(t *testing.T) {
	s, _, mockRedis, ctx := setup(t)
	expectedUuid := uuid.New()

	mockRedis.EXPECT().
		ChangeStatusByRouterId(ctx, expectedUuid, "SENT").
		Return(fmt.Errorf("wrong type of query")).Times(1)

	err := s.ChangeStatus(ctx, expectedUuid, "SENT")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to change command status in Redis:")
}

func TestChangeStatus_ErrorInPostgres(t *testing.T) {
	s, mockPostgres, mockRedis, ctx := setup(t)
	expectedUuid := uuid.New()

	mockRedis.EXPECT().
		ChangeStatusByRouterId(ctx, expectedUuid, gomock.Eq("SENT")).
		Return(nil).Times(1)

	mockPostgres.EXPECT().
		ChangeStatusByRouterId(ctx, expectedUuid, gomock.Eq("SENT")).
		Return(fmt.Errorf("wrong type of query")).Times(1)

	err := s.ChangeStatus(ctx, expectedUuid, "SENT")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to change command status in DB:")
}
