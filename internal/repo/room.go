package repo

import (
	"context"

	"github.com/google/uuid"
)

type Room struct {
	ID          uuid.UUID
	Name        string
	Description *string
	Capacity    *int
}

func (r *Repo) CreateRoom(ctx context.Context, name string, description *string, capacity *int) (*Room, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO rooms (name, description, capacity)
         VALUES ($1, $2, $3)
         RETURNING id, name, description, capacity`,
		name, description, capacity,
	)

	var room Room
	if err := row.Scan(&room.ID, &room.Name, &room.Description, &room.Capacity); err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *Repo) ListRooms(ctx context.Context) ([]Room, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, description, capacity
         FROM rooms
         ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Room
	for rows.Next() {
		var room Room
		if err := rows.Scan(&room.ID, &room.Name, &room.Description, &room.Capacity); err != nil {
			return nil, err
		}
		res = append(res, room)
	}
	return res, rows.Err()
}
