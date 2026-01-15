package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"router-manager/internal/metrics"
	"router-manager/internal/model"
	"router-manager/internal/pb"
	"router-manager/internal/repository/postgres"
	"router-manager/internal/repository/redis"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type CommandService struct {
	pb.UnimplementedCommandServiceServer

	redisRepo    redis.RedisRepo
	postgresRepo postgres.PostgresRepo
}

func NewCommandService(pgRepo postgres.PostgresRepo, redisRepo redis.RedisRepo) *CommandService {
	return &CommandService{
		postgresRepo: pgRepo,
		redisRepo:    redisRepo,
	}
}

func (s *CommandService) SendCommand(ctx context.Context, req *pb.SendCommandRequest) (*pb.SendCommandResponse, error) {
	// metrics initialization
	metrics.SendCommandCalls.Inc()

	timer := prometheus.NewTimer(metrics.SendCommandHistogramm)
	defer timer.ObserveDuration()

	if len(req.Routers) == 0 {
		return nil, fmt.Errorf("no routers specified")
	}
	if req.CommandType == "" {
		return nil, fmt.Errorf("no command specified")
	}

	var commandsIds []string
	for _, routers := range req.Routers {
		if routers.SerialNumber == "" {
			return nil, fmt.Errorf("router serial_number is required")
		}

		log.Printf("Sending command to router %s", routers.SerialNumber)

		now := time.Now()
		router := &model.Router{
			ID:           uuid.New(),
			SerialNumber: routers.SerialNumber,
			LastSeenAt:   &now,
			CreatedAt:    now,
			IPAddress:    nil,
		}
		s.SaveRouter(ctx, router)

		cmd := &model.Command{
			ID:          uuid.New(),
			RouterID:    router.ID,
			CommandType: req.CommandType,
			Payload:     json.RawMessage(fmt.Sprintf(`{"command": "%s"}`, req.CommandType)),
			Status:      "PENDING",
			CreatedAt:   time.Now(),
		}

		if err := s.postgresRepo.SaveCommand(ctx, cmd); err != nil {
			return nil, fmt.Errorf("failed to save command in PostgreSQL: %w", err)
		}

		if err := s.redisRepo.SaveCommand(ctx, cmd); err != nil {
			fmt.Printf("Warning: failed to save command in Redis: %v\n", err)
		}

		commandsIds = append(commandsIds, cmd.ID.String())
	}

	log.Printf("Commands sent.")

	return &pb.SendCommandResponse{
		Status: "PENDING",
		Id:     commandsIds,
	}, nil
}

func (s *CommandService) PollCommands(ctx context.Context, req *pb.PollRequest) (*pb.PollResponse, error) {
	// metrics initialization
	metrics.CommadsPollCalls.Inc()

	timer := prometheus.NewTimer(metrics.CommandsPollHistogramm)
	defer timer.ObserveDuration()

	if req.SerialNumber == "" {
		return nil, fmt.Errorf("router serial_number is required")
	}

	router := s.findRouter(ctx, req.RouterId)
	if router == nil {
		return nil, fmt.Errorf("there is no router with such serial_number: %s", req.SerialNumber)
	}

	log.Printf("Poll commands for router %s", req.RouterId)

	now := time.Now()
	router.LastSeenAt = &now

	s.SaveRouter(ctx, router)

	var commands []model.Command
	commands, err := s.redisRepo.FindCommandsByRouterId(ctx, router.ID)
	if err != nil {
		log.Printf("Failed to get commands from Redis: %v", err)
	}

	if len(commands) == 0 {
		commands, err = s.postgresRepo.GetCommandsByRouterId(ctx, router.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load commands from DB: %w", err)
		}
	}

	var pbCommandsResponse []*pb.Command
	for _, command := range commands {
		pbCommandsResponse = append(pbCommandsResponse, &pb.Command{
			Id:          command.ID.String(),
			CommandType: command.CommandType,
			Payload:     string(command.Payload),
			CreatedAt:   timestamppb.New(command.CreatedAt),
		})
	}

	err = s.ChangeStatus(ctx, router.ID, "SENT")
	if err != nil {
		return nil, err
	}

	log.Printf("Commands sent to router.")

	return &pb.PollResponse{
		Commands: pbCommandsResponse,
	}, nil
}

func (s *CommandService) AckCommand(ctx context.Context, req *pb.AckRequest) (*pb.AckResponse, error) {
	// metrics initialization
	metrics.CommandsAckCalls.Inc()

	timer := prometheus.NewTimer(metrics.CommandsAckHistogramm)
	defer timer.ObserveDuration()

	if req.SerialNumber == "" {
		return nil, fmt.Errorf("serial_number is required")
	}
	if req.CommandType == "" {
		return nil, fmt.Errorf("command_type is required")
	}

	router := s.findRouter(ctx, req.RouterId)
	if router == nil {
		return nil, fmt.Errorf("there is no router with such serial_number: %s", req.SerialNumber)
	}

	log.Printf("Ack commands for router %s", req.RouterId)

	now := time.Now()
	router.LastSeenAt = &now

	s.SaveRouter(ctx, router)

	err := s.ChangeStatus(ctx, router.ID, "ACKED")
	if err != nil {
		return nil, err
	}

	log.Printf("Commands acked.")

	return &pb.AckResponse{
		Status: "ACKED",
	}, nil
}

func (s *CommandService) findRouter(ctx context.Context, id string) *model.Router {
	// check if we've already had this router
	router, err := s.redisRepo.FindRouterByRouterId(ctx, id)
	if err != nil {
		fmt.Printf("Redis lookup failed for %s: %v\n", id, err)
	}

	// check in Postgres
	if router == nil {
		router, err = s.postgresRepo.FindRouterByRouterId(ctx, id)
		if err != nil {
			fmt.Printf("Router not found in DB: %v\n", err)
		}
	}

	return router
}

func (s *CommandService) SaveRouter(ctx context.Context, router *model.Router) {
	if err := s.postgresRepo.SaveRouter(ctx, router); err != nil {
		log.Printf("ERROR: failed to save router in PostgreSQL: %v", err)
		return
	}

	// Redis — опционально
	if err := s.redisRepo.SaveRouter(ctx, router); err != nil {
		log.Printf("WARNING: failed to save router in Redis: %v", err)
	}
}

func (s *CommandService) ChangeStatus(ctx context.Context, routerId uuid.UUID, status string) error {
	err := s.redisRepo.ChangeStatusByRouterId(ctx, routerId, status)
	if err != nil {
		return fmt.Errorf("failed to change command status in Redis: %w", err)
	}

	err = s.postgresRepo.ChangeStatusByRouterId(ctx, routerId, status)
	if err != nil {
		return fmt.Errorf("failed to change command status in DB: %w", err)
	}

	return nil
}
