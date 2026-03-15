package model

import "time"

type OutboxEntry struct {
	ID        int64     `db:"id"`
	Topic     string    `db:"topic"`
	EventType string    `db:"event_type"`
	Payload   []byte    `db:"payload"`
	Published bool      `db:"published"`
	CreatedAt time.Time `db:"created_at"`
}
