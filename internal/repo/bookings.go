package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SlotRecord struct {
	ID      uuid.UUID
	RoomID  uuid.UUID
	StartAt time.Time
	EndAt   time.Time
	Status  string
}

type Booking struct {
	ID         uuid.UUID
	SlotID     uuid.UUID
	UserID     uuid.UUID
	Status     string
	CreatedAt  time.Time
	CancelledAt *time.Time
}

func (r *Repo) FindOrCreateSlot(
	ctx context.Context,
	roomID uuid.UUID,
	start, end time.Time,
) (*SlotRecord, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO slots (room_id, start_at, end_at, status)
		VALUES ($1, $2, $3, 'FREE')
		ON CONFLICT (room_id, start_at, end_at) DO UPDATE SET room_id = EXCLUDED.room_id
		RETURNING id, room_id, start_at, end_at, status
	`, roomID, start, end)

	var s SlotRecord
	if err := row.Scan(&s.ID, &s.RoomID, &s.StartAt, &s.EndAt, &s.Status); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) BookSlot(
	ctx context.Context,
	slotID, userID uuid.UUID,
) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx,
		`SELECT status FROM slots WHERE id = $1 FOR UPDATE`,
		slotID,
	).Scan(&status)
	if err != nil {
		return nil, err
	}
	if status != "FREE" {
		return nil, ErrSlotNotFree
	}

	var b Booking
	err = tx.QueryRowContext(ctx, `
		INSERT INTO bookings (slot_id, user_id, status)
		VALUES ($1, $2, 'ACTIVE')
		RETURNING id, slot_id, user_id, status, created_at, cancelled_at
	`, slotID, userID).Scan(&b.ID, &b.SlotID, &b.UserID, &b.Status, &b.CreatedAt, &b.CancelledAt)
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE slots SET status = 'BOOKED' WHERE id = $1`,
		slotID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &b, nil
}
