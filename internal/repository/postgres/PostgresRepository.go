package postgres

import (
	"context"
	"fmt"
	"router-manager/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepo interface {
	SaveCommand(ctx context.Context, cmd *model.Command) error
	GetCommandsByRouterId(ctx context.Context, routerId uuid.UUID) ([]model.Command, error)
	ChangeStatusByRouterId(ctx context.Context, routerId uuid.UUID, status string) error
	SaveRouter(ctx context.Context, router *model.Router) error
	FindRouterByRouterId(ctx context.Context, id string) (*model.Router, error)
}

type PostgresRepository struct {
	pool *pgxpool.Pool
}

/* --- work with commmands table --- */

func NewPostgresRepository(pool *pgxpool.Pool) PostgresRepo {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) SaveCommand(ctx context.Context, cmd *model.Command) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO commands (
			id, router_id, command_type, payload, status, sent_at, acked_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			acked_at = EXCLUDED.acked_at,
			sent_at = COALESCE(EXCLUDED.sent_at, commands.sent_at)`,
		cmd.ID,
		cmd.RouterID,
		cmd.CommandType,
		cmd.Payload,
		cmd.Status,
		cmd.SentAt,
		cmd.AckedAt,
		cmd.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetCommandsByRouterId(ctx context.Context, routerId uuid.UUID) ([]model.Command, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT 
			id, router_id, command_type, payload, 
			status, sent_at, acked_at, created_at
		FROM commands 
		WHERE router_id = $1 
		ORDER BY created_at ASC`,
		routerId)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commands []model.Command
	for rows.Next() {
		var cmd model.Command
		err := rows.Scan(
			&cmd.ID,
			&cmd.RouterID,
			&cmd.CommandType,
			&cmd.Payload,
			&cmd.Status,
			&cmd.SentAt,
			&cmd.AckedAt,
			&cmd.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan command row: %w", err)
		}
		commands = append(commands, cmd)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return commands, nil
}

func (r *PostgresRepository) ChangeStatusByRouterId(ctx context.Context, routerId uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE commands
        SET status = $1,
            sent_at = CASE
                WHEN $1 = 'SENT' AND status = 'PENDING' THEN NOW()
                ELSE sent_at
            END,
            acked_at = CASE
                WHEN $1 = 'ACKED' AND status ='SENT' THEN NOW()
                ELSE acked_at
            END
        WHERE router_id = $2`,
		status, routerId)

	if err != nil {
		return err
	}

	return nil
}

/* --- work with routers table --- */

func (r *PostgresRepository) SaveRouter(ctx context.Context, router *model.Router) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO routers (id, serial_number, ip_address, last_seen_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (serial_number) DO UPDATE SET
			ip_address = EXCLUDED.ip_address,
			last_seen_at = EXCLUDED.last_seen_at`,
		router.ID,
		router.SerialNumber,
		router.IPAddress,
		router.LastSeenAt,
		router.CreatedAt)
	return err
}

func (r *PostgresRepository) FindRouterByRouterId(ctx context.Context, id string) (*model.Router, error) {
	var router model.Router
	err := r.pool.QueryRow(ctx,
		`SELECT id, serial_number, ip_address, last_seen_at, created_at
		FROM routers
		WHERE id = $1`,
		id).Scan(
		&router.ID,
		&router.SerialNumber,
		&router.IPAddress,
		&router.LastSeenAt,
		&router.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &router, nil
}
