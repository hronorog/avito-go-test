package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hronorog/avito-go-test/internal/repo"
)

var (
    ErrSlotAlreadyBooked = errors.New("slot already booked")
    ErrSlotInPast        = errors.New("slot in past")
    ErrSlotNotFound      = errors.New("slot not found")
	ErrBookingNotFound   = errors.New("booking not found")
	ErrForbiddenBooking  = errors.New("forbidden booking")
)

func (s *Service) CreateBooking(ctx context.Context, roomID, userID uuid.UUID, start, end time.Time,) (*repo.Booking, error) {
	// проверяем длительность 30 минут
	if !start.Before(end) || end.Sub(start) != 30*time.Minute {
		return nil, ErrInvalidRequest
	}
	// в прошлом нельзя
	if end.Before(time.Now()) {
		return nil, ErrSlotInPast
	}

	slot, err := s.repo.FindOrCreateSlot(ctx, roomID, start, end)
	if err != nil {
		return nil, err
	}
	if slot.Status != "FREE" {
		return nil, ErrSlotAlreadyBooked
	}

	b, err := s.repo.BookSlot(ctx, slot.ID, userID)
	if err != nil {
		if errors.Is(err, repo.ErrSlotNotFree) {
			return nil, ErrSlotAlreadyBooked
		}
		return nil, err
	}
	return b, nil
}

func (s *Service) CreateBookingBySlotID(ctx context.Context, slotID, userID uuid.UUID,) (*repo.Booking, uuid.UUID, error) {
	slot, err := s.repo.GetSlotByID(ctx, slotID)
	if err != nil {
		if errors.Is(err, repo.ErrSlotNotFound) {
			return nil, uuid.Nil, ErrSlotNotFound
		}
		return nil, uuid.Nil, err
	}

	if slot.StartAt.Before(time.Now()) {
		return nil, uuid.Nil, ErrSlotInPast
	}

	b, err := s.repo.BookExistingSlot(ctx, slot.ID, userID)
	if err != nil {
		switch {
		case errors.Is(err, repo.ErrSlotNotFree):
			return nil, uuid.Nil, ErrSlotAlreadyBooked
		case errors.Is(err, repo.ErrSlotNotFound):
			return nil, uuid.Nil, ErrSlotNotFound
		default:
			return nil, uuid.Nil, err
		}
	}

	return b, slot.RoomID, nil
}

func (s *Service) ListMyBookings(ctx context.Context, userID uuid.UUID) ([]repo.BookingWithSlot, error) {
	return s.repo.ListUserFutureBookings(ctx, userID)
}

func (s *Service) CancelMyBooking(ctx context.Context, bookingID, userID uuid.UUID) (*repo.Booking, error) {
	b, err := s.repo.GetBookingByID(ctx, bookingID)
	if err != nil {
		if errors.Is(err, repo.ErrBookingNotFound) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}

	if b.UserID != userID {
		return nil, ErrForbiddenBooking
	}

	cb, err := s.repo.CancelBooking(ctx, bookingID)
	if err != nil {
		if errors.Is(err, repo.ErrBookingNotFound) {
			return nil, ErrBookingNotFound
		}
		return nil, err
	}
	return cb, nil
}

func (s *Service) ListAllBookings(ctx context.Context, page, pageSize int) ([]repo.Booking, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	return s.repo.ListAllBookings(ctx, pageSize, offset)
}
