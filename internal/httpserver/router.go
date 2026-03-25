package httpserver

import (
	// debug
	"log"
	//
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/hronorog/avito-go-test/internal/auth"
	"github.com/hronorog/avito-go-test/internal/repo"
	"github.com/hronorog/avito-go-test/internal/service"

	"github.com/google/uuid"
)


type RoomDTO struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Capacity    *int    `json:"capacity,omitempty"`
}

type CreateRoomRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Capacity    *int    `json:"capacity"`
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
	ID         string `json:"id"`
	RoomID     string `json:"roomId"`
	DaysOfWeek []int  `json:"daysOfWeek"`
	StartTime  string `json:"startTime"` // "HH:MM"
	EndTime    string `json:"endTime"`   // "HH:MM"
}

type CreateScheduleRequest struct {
	DaysOfWeek []int  `json:"daysOfWeek"`
	StartTime  string `json:"startTime"` // "HH:MM"
	EndTime    string `json:"endTime"`   // "HH:MM"
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
		if len(parts) == 3 && parts[1] == "schedule" && parts[2] == "create" {
			roomIDStr := parts[0]
			scheduleCreateHandler(w, r, s, roomIDStr)
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
