package httpserver

import (
	// debug
	"log"
	//
	"database/sql"
	"encoding/json"
	"net/http"
	// "net/url"
	"strings"
	"time"

	"github.com/hronorog/avito-go-test/internal/auth"
	"github.com/hronorog/avito-go-test/internal/repo"
	"github.com/hronorog/avito-go-test/internal/service"

	"github.com/google/uuid"
)


type RoomDTO struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description *string  `json:"description,omitempty"`
	Capacity    *int     `json:"capacity,omitempty"`
}

type SlotDTO struct {
	StartTime  string  `json:"startTime"`
	EndTime    string  `json:"endTime"`
	Status     string  `json:"status"`
	BookingID *string  `json:"bookingId,omitempty"`
}

type SlotsListResponse struct {
	Slots []SlotDTO `json:"slots"`
}

type CreateRoomRequest struct {
	Name         string  `json:"name"`
	Description *string  `json:"description"`
	Capacity    *int     `json:"capacity"`
}

type DummyLoginRequest struct {
	Role string `json:"role"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ScheduleDTO struct {
	ID           string `json:"id"`
	RoomID       string `json:"roomId"`
	DaysOfWeek []int    `json:"daysOfWeek"`
	StartTime    string `json:"startTime"` // "HH:MM"
	EndTime      string `json:"endTime"`   // "HH:MM"
}

type CreateScheduleRequest struct {
	DaysOfWeek []int    `json:"daysOfWeek"`
	StartTime    string `json:"startTime"` // "HH:MM"
	EndTime      string `json:"endTime"`   // "HH:MM"
}

type CreateBookingRequest struct {
	SlotID              string `json:"slotId"`
	CreateConferenceLink bool   `json:"createConferenceLink"`
}

type BookingDTO struct {
	ID             string  `json:"id"`
	SlotID         string  `json:"slotId"`
	UserID         string  `json:"userId"`
	Status         string  `json:"status"`
	ConferenceLink *string `json:"conferenceLink"`
	CreatedAt      string  `json:"createdAt"`
}

func New(db *sql.DB) http.Handler {
	rp := repo.New(db)
	s := service.New(rp)

	mux := http.NewServeMux()

	mux.HandleFunc("/_info", func(w http.ResponseWriter, r *http.Request) {
		if err := s.Health(r.Context()); err != nil {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "db not available")
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/dummyLogin", dummyLoginHandler)

	mux.HandleFunc("/rooms/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/rooms/")
		parts := strings.Split(path, "/")

		// /rooms/{roomId}/schedule/create
		if len(parts) == 3 && parts[1] == "schedule" && parts[2] == "create" {
			roomIDStr := parts[0]
			scheduleCreateHandler(w, r, s, roomIDStr)
			return
		}

		// /rooms/{roomId}/slots/list
		if len(parts) == 3 && parts[1] == "slots" && parts[2] == "list" {
			roomIDStr := parts[0]
			slotsListHandler(w, r, s, roomIDStr)
			return
		}

		http.NotFound(w, r)
	})

	mux.HandleFunc("/rooms/create", func(w http.ResponseWriter, r *http.Request) {
		createRoomHandler(w, r, s)
	})
	mux.HandleFunc("/rooms/list", func(w http.ResponseWriter, r *http.Request) {
		listRoomsHandler(w, r, s)
	})

	mux.HandleFunc("/bookings/create", func(w http.ResponseWriter, r *http.Request) {
		createBookingHandler(w, r, s)
	})

	return auth.Middleware(mux)
}

func dummyLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req DummyLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}

	if req.Role != "admin" && req.Role != "user" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "role must be 'admin' or 'user'")
		return
	}

	token, _, err := auth.GenerateToken(req.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate token")
		return
	}

	resp := TokenResponse{Token: token}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Code:    code,
		Message: msg,
	})
}

func roomToDTO(r repo.Room) RoomDTO {
	return RoomDTO{
		ID:          r.ID.String(),
		Name:        r.Name,
		Description: r.Description,
		Capacity:    r.Capacity,
	}
}

func createRoomHandler(w http.ResponseWriter, r *http.Request, s *service.Service) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		return
	}

	var req CreateRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
		return
	}

	room, err := s.CreateRoom(r.Context(), req.Name, req.Description, req.Capacity)
	if err != nil {
		if err == service.ErrInvalidRequest {
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create room")
		return
	}

	resp := struct {
		Room RoomDTO `json:"room"`
	}{
		Room: roomToDTO(*room),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func listRoomsHandler(w http.ResponseWriter, r *http.Request, s *service.Service) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	_ = user // достаточно того, что он есть (admin или user)

	rooms, err := s.ListRooms(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list rooms")
		return
	}

	dtos := make([]RoomDTO, 0, len(rooms))
	for _, rm := range rooms {
		dtos = append(dtos, roomToDTO(rm))
	}

	resp := struct {
		Rooms []RoomDTO `json:"rooms"`
	}{
		Rooms: dtos,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func scheduleToDTO(s repo.Schedule) ScheduleDTO {
	return ScheduleDTO{
		ID:         s.ID.String(),
		RoomID:     s.RoomID.String(),
		DaysOfWeek: s.DaysOfWeek,
		StartTime:  s.StartTime.Format("15:04"),
		EndTime:    s.EndTime.Format("15:04"),
	}
}

func scheduleCreateHandler(w http.ResponseWriter, r *http.Request, s *service.Service, roomIDStr string) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	if user.Role != "admin" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "admin role required")
		return
	}

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid room id")
		return
	}

	var req CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}

	start, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid startTime")
		return
	}
	end, err := time.Parse("15:04", req.EndTime)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid endTime")
		return
	}

	schedule, err := s.CreateSchedule(r.Context(), roomID, req.DaysOfWeek, start, end)
	if err != nil {
		// debug
		log.Printf("CreateSchedule error: %v", err)
		//
		switch err {
		case service.ErrInvalidRequest:
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
			return
		case service.ErrScheduleExists:
			writeError(w, http.StatusConflict, "SCHEDULE_EXISTS", "schedule already exists")
			return
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create schedule")
			return
		}
	}

	resp := struct {
		Schedule ScheduleDTO `json:"schedule"`
	}{
		Schedule: scheduleToDTO(*schedule),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func slotsListHandler(w http.ResponseWriter, r *http.Request, s *service.Service, roomIDStr string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing auth")
		return
	}
	_ = user 

	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid room id")
		return
	}

	q := r.URL.Query()
	dateStr := q.Get("date")
	if dateStr == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "date is required")
		return
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid date")
		return
	}

	slots, err := s.ListSlotsForRoomDate(r.Context(), roomID, date)
	if err != nil {
		log.Printf("ListSlotsForRoomDate error: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list slots")
		return
	}

	dtos := make([]SlotDTO, 0, len(slots))
	for _, sl := range slots {
		dtos = append(dtos, SlotDTO{
			StartTime: sl.Start.UTC().Format(time.RFC3339),
			EndTime:   sl.End.UTC().Format(time.RFC3339),
			Status:    "FREE",
			BookingID: nil,
		})
	}

	resp := SlotsListResponse{Slots: dtos}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func bookingToDTO(b *repo.Booking) BookingDTO {
	var cancelled *string
	if b.CancelledAt != nil {
		s := b.CancelledAt.UTC().Format(time.RFC3339)
		cancelled = &s
	}
	_ = cancelled 

	return BookingDTO{
		ID:             b.ID.String(),
		SlotID:         b.SlotID.String(),
		UserID:         b.UserID.String(),
		Status:         b.Status,        
		ConferenceLink: nil,             
		CreatedAt:      b.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func createBookingHandler(w http.ResponseWriter, r *http.Request, s *service.Service) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	user, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	if user.Role != "user" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "booking allowed only for user role")
		return
	}

	var req CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}
	if req.SlotID == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "slotId is required")
		return
	}

	slotID, err := uuid.Parse(req.SlotID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid slotId")
		return
	}

	userID, err := uuid.Parse(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "invalid user id in token")
		return
	}

	booking, _, err := s.CreateBookingBySlotID(r.Context(), slotID, userID)
	if err != nil {
		switch err {
		case service.ErrSlotInPast:
			writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "slot in past")
			return
		case service.ErrSlotAlreadyBooked:
			writeError(w, http.StatusConflict, "SLOT_ALREADY_BOOKED", "slot is already booked")
			return
		case service.ErrSlotNotFound:
			writeError(w, http.StatusNotFound, "SLOT_NOT_FOUND", "slot not found")
			return
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create booking")
			return
		}
	}

	dto := bookingToDTO(booking)
	resp := struct {
		Booking BookingDTO `json:"booking"`
	}{
		Booking: dto,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
