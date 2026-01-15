package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Command struct {
	ID          uuid.UUID       `db:"id"`
	RouterID    uuid.UUID       `db:"router_id"`
	CommandType string          `db:"command_type"`
	Payload     json.RawMessage `db:"payload"`
	Status      string          `db:"status"`
	SentAt      *time.Time      `db:"sent_at"`
	AckedAt     *time.Time      `db:"acked_at"`
	CreatedAt   time.Time       `db:"created_at"`
}
