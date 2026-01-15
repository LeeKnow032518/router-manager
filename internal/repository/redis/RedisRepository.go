package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"router-manager/internal/model"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisRepo interface {
	SaveCommand(ctx context.Context, command *model.Command) error
	FindCommandsByRouterId(ctx context.Context, routerId uuid.UUID) ([]model.Command, error)
	ChangeStatusByRouterId(ctx context.Context, routerId uuid.UUID, status string) error
	SaveRouter(ctx context.Context, router *model.Router) error
	FindRouterByRouterId(ctx context.Context, id string) (*model.Router, error)
}

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) RedisRepo {
	return &RedisRepository{client: client}
}

/* --- work with commmands table --- */

func (r *RedisRepository) SaveCommand(ctx context.Context, command *model.Command) error {
	data, err := json.Marshal(command)
	if err != nil {
		return err
	}

	key := "command:" + command.RouterID.String()
	_, err = r.client.RPush(ctx, key, data).Result()
	return err
}

func (r *RedisRepository) FindCommandsByRouterId(ctx context.Context, routerId uuid.UUID) ([]model.Command, error) {
	key := fmt.Sprintf("command:%s", routerId.String())
	values, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get commands from Redis: %w", err)
	}

	var commands []model.Command
	for _, v := range values {
		var command model.Command
		if err := json.Unmarshal([]byte(v), &command); err != nil {
			return nil, fmt.Errorf("failed to unmarshal command: %w", err)
		}
		commands = append(commands, command)
	}

	return commands, nil
}

func (r *RedisRepository) ChangeStatusByRouterId(ctx context.Context, routerId uuid.UUID, status string) error {
	key := fmt.Sprintf("command:%s", routerId.String())
	values, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get commands from Redis: %w", err)
	}

	if len(values) == 0 {
		return nil
	}

	var updated []string
	for _, v := range values {
		var cmd model.Command
		if err := json.Unmarshal([]byte(v), &cmd); err != nil {
			return fmt.Errorf("failed to unmarshal command: %w", err)
		}

		now := time.Now()
		if status == "SENT" && cmd.Status == "PENDING" {
			cmd.SentAt = &now
		} else if status == "ACKED" && cmd.Status == "SENT" {
			cmd.AckedAt = &now
		} else {
			return fmt.Errorf("wrong type of query")
		}

		cmd.Status = status

		data, err := json.Marshal(cmd)
		if err != nil {
			updated = append(updated, v)
			continue
		}
		updated = append(updated, string(data))
	}

	pipe := r.client.TxPipeline()
	pipe.Del(context.Background(), key)
	if len(updated) > 0 {
		pipe.RPush(context.Background(), key, updated)
	}
	_, err = pipe.Exec(context.Background())
	if err != nil {
		return fmt.Errorf("failed to update Redis: %w", err)
	}

	return nil
}

/* --- work with routers table --- */

func (r *RedisRepository) SaveRouter(ctx context.Context, router *model.Router) error {
	data, err := json.Marshal(router)
	if err != nil {
		return err
	}

	key := "router:" + router.ID.String()
	_, err = r.client.Set(ctx, key, data, 24*time.Hour).Result()
	return err
}

func (r *RedisRepository) FindRouterByRouterId(ctx context.Context, id string) (*model.Router, error) {
	key := "router:" + id
	data, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}

	var router model.Router
	if err := json.Unmarshal([]byte(data), &router); err != nil {
		return nil, err
	}

	return &router, nil
}
