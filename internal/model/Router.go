package model

import (
	"net"
	"time"

	"github.com/google/uuid"
)

type Router struct {
	ID           uuid.UUID  `db:"id"`
	SerialNumber string     `db:"serial_number"`
	IPAddress    net.IP     `db:"ip_address"`
	LastSeenAt   *time.Time `db:"last_seen_at"`
	CreatedAt    time.Time  `db:"created_at"`
}
