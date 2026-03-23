package httpserver

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/hronorog/avito-go-test/internal/auth"
	"github.com/hronorog/avito-go-test/internal/repo"
	"github.com/hronorog/avito-go-test/internal/service"
)

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

func New(db *sql.DB) http.Handler {
	r := repo.New(db)
	s := service.New(r)

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
