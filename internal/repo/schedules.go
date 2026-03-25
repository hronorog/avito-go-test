package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Schedule struct {
	ID         uuid.UUID
	RoomID     uuid.UUID
	DaysOfWeek []int
	StartTime  time.Time 
	EndTime    time.Time
}

func (r *Repo) CreateSchedule(ctx context.Context, roomID uuid.UUID, days []int, start, end time.Time) (*Schedule, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO room_schedules (room_id, days_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4)
		RETURNING id, room_id, days_of_week, start_time, end_time
	`, roomID, pq.Array(days), start.Format("15:04:05"), end.Format("15:04:05"))

	var s Schedule
	var startStr, endStr string
	if err := row.Scan(&s.ID, &s.RoomID, pq.Array(&s.DaysOfWeek), &startStr, &endStr); err != nil {
		return nil, err
	}

	today := time.Now().Truncate(24 * time.Hour)
	st, _ := time.Parse("15:04:05", startStr)
	et, _ := time.Parse("15:04:05", endStr)
	s.StartTime = time.Date(today.Year(), today.Month(), today.Day(), st.Hour(), st.Minute(), st.Second(), 0, time.UTC)
	s.EndTime = time.Date(today.Year(), today.Month(), today.Day(), et.Hour(), et.Minute(), et.Second(), 0, time.UTC)

	return &s, nil
}
