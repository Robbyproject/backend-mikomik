package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"mikomik-backend/db"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// jwtSecret — in production put this in an env var.
var jwtSecret = []byte("mikomik-super-secret-key-change-me")

type authReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserResp struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

// ---------- helpers ----------

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func setTokenCookie(w http.ResponseWriter, userID int64, role string) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(72 * time.Hour).Unix(),
	}
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)

	http.SetCookie(w, &http.Cookie{
		Name:     "mikomik_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3 * 24 * 3600,
	})
}

// ParseToken extracts user ID and role from the JWT cookie.
func ParseToken(r *http.Request) (int64, string, bool) {
	c, err := r.Cookie("mikomik_token")
	if err != nil {
		return 0, "", false
	}
	token, err := jwt.Parse(c.Value, func(t *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, "", false
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, "", false
	}
	id, ok1 := claims["sub"].(float64)
	role, ok2 := claims["role"].(string)
	if !ok1 || !ok2 {
		return 0, "", false
	}
	return int64(id), role, true
}

// ---------- handlers ----------

// Register creates a new user account.
func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonErr(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	var req authReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if req.Username == "" || req.Email == "" || req.Password == "" {
		jsonErr(w, "username, email and password are required", http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		jsonErr(w, "password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}

	// First user gets admin role
	var count int
	db.Conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	role := "user"
	if count == 0 {
		role = "admin"
	}

	res, err := db.Conn.Exec(
		"INSERT INTO users (username, email, password, role) VALUES (?, ?, ?, ?)",
		req.Username, req.Email, string(hash), role,
	)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			jsonErr(w, "username or email already exists", http.StatusConflict)
			return
		}
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	setTokenCookie(w, id, role)

	jsonOK(w, map[string]any{
		"message": "registered",
		"user":    UserResp{ID: id, Username: req.Username, Email: req.Email, Role: role, CreatedAt: time.Now().Format(time.RFC3339)},
	})
}

// Login authenticates an existing user.
func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonErr(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}

	var req authReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonErr(w, "invalid body", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		jsonErr(w, "username and password are required", http.StatusBadRequest)
		return
	}

	var id int64
	var hash, email, avatar, role string
	var createdAt time.Time
	err := db.Conn.QueryRow(
		"SELECT id, password, email, COALESCE(avatar, ''), role, created_at FROM users WHERE username = ?",
		req.Username,
	).Scan(&id, &hash, &email, &avatar, &role, &createdAt)
	if err == sql.ErrNoRows {
		jsonErr(w, "invalid username or password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
		jsonErr(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	setTokenCookie(w, id, role)

	jsonOK(w, map[string]any{
		"message": "logged in",
		"user":    UserResp{ID: id, Username: req.Username, Email: email, Avatar: avatar, Role: role, CreatedAt: createdAt.Format(time.RFC3339)},
	})
}

// Me returns the current authenticated user.
func Me(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	id, _, ok := ParseToken(r)
	if !ok {
		jsonErr(w, "not authenticated", http.StatusUnauthorized)
		return
	}

	var u UserResp
	var createdAt time.Time
	err := db.Conn.QueryRow(
		"SELECT id, username, email, COALESCE(avatar, ''), role, created_at FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Username, &u.Email, &u.Avatar, &u.Role, &createdAt)
	if err != nil {
		jsonErr(w, "user not found", http.StatusNotFound)
		return
	}
	u.CreatedAt = createdAt.Format(time.RFC3339)

	jsonOK(w, map[string]any{"user": u})
}

// Logout clears the auth cookie.
func Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "mikomik_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	jsonOK(w, map[string]string{"message": "logged out"})
}
