package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Slot struct {
	Start time.Time
	End   time.Time
}

func (s *Service) ListSlotsForRoomDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]Slot, error) {
	sched, err := s.repo.GetScheduleByRoomID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	allowed := false
	for _, d := range sched.DaysOfWeek {
		if int(d) == weekday {
			allowed = true
			break
		}
	}
	if !allowed {
		return []Slot{}, nil
	}

	year, month, day := date.Date()
	loc := time.UTC

	start := time.Date(year, month, day,
		sched.StartTime.Hour(), sched.StartTime.Minute(), 0, 0, loc)
	end := time.Date(year, month, day,
		sched.EndTime.Hour(), sched.EndTime.Minute(), 0, 0, loc)

	var slots []Slot
	cur := start
	for cur.Add(30 * time.Minute).Before(end) || cur.Add(30*time.Minute).Equal(end) {
		slotEnd := cur.Add(30 * time.Minute)
		slots = append(slots, Slot{
			Start: cur,
			End:   slotEnd,
		})
		cur = slotEnd
	}

	return slots, nil
}
