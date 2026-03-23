package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID   string
	Role string
}

type contextKey string

const userKey contextKey = "user"

const (
	AdminUserID = "11111111-1111-1111-1111-111111111111"
	NormalUserID = "22222222-2222-2222-2222-222222222222"
)

func secret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "dev-secret"
	}
	return []byte(s)
}

func GenerateToken(role string) (string, string, error) {
	var userID string
	switch role {
	case "admin":
		userID = AdminUserID
	case "user":
		userID = NormalUserID
	default:
		return "", "", errors.New("invalid role")
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret())
	if err != nil {
		return "", "", err
	}

	return signed, userID, nil
}

func ParseToken(tokenStr string) (*User, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret(), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}

	id, _ := claims["user_id"].(string)
	role, _ := claims["role"].(string)

	if id == "" || role == "" {
		return nil, errors.New("missing claims")
	}

	return &User{ID: id, Role: role}, nil
}

func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

func FromContext(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(userKey).(*User)
	return u, ok
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			next.ServeHTTP(w, r)
			return
		}

		user, err := ParseToken(parts[1])
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := WithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
