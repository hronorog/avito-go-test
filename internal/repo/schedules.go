package repo

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Schedule struct {
    ID         uuid.UUID
    RoomID     uuid.UUID
    DaysOfWeek pq.Int64Array
    StartTime  time.Time
    EndTime    time.Time
}

func (r *Repo) CreateSchedule(ctx context.Context, roomID uuid.UUID, days []int, start, end time.Time,) (*Schedule, error) {
    daysPg := make(pq.Int64Array, len(days))
    for i, d := range days {
        daysPg[i] = int64(d)
    }

    row := r.db.QueryRowContext(ctx, `
        INSERT INTO schedules (room_id, days_of_week, start_time, end_time)
        VALUES ($1, $2, $3, $4)
        RETURNING id, room_id, days_of_week, start_time, end_time
    `, roomID, daysPg, start, end)

    var s Schedule
    if err := row.Scan(
        &s.ID,
        &s.RoomID,
        &s.DaysOfWeek,
        &s.StartTime,
        &s.EndTime,
    ); err != nil {
        return nil, err
    }
    return &s, nil
}

func (r *Repo) GetScheduleByRoomID(ctx context.Context, roomID uuid.UUID) (*Schedule, error) {
    row := r.db.QueryRowContext(ctx, `
        SELECT id, room_id, days_of_week, start_time, end_time
        FROM schedules
        WHERE room_id = $1
    `, roomID)

    var s Schedule
    if err := row.Scan(
        &s.ID,
        &s.RoomID,
        &s.DaysOfWeek,
        &s.StartTime,
        &s.EndTime,
    ); err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return &s, nil
}
