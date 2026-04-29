package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"mikomik-backend/db"
)

// ── Profile endpoints ────────────────────────────────

// GetProfile returns current user's profile. GET /api/user/profile
func GetProfile(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	userID, _, ok := ParseToken(r)
	if !ok {
		jsonErr(w, "login required", http.StatusUnauthorized)
		return
	}
	var username, email, avatar, role string
	err := db.Conn.QueryRow("SELECT username, email, COALESCE(avatar,''), role FROM users WHERE id = ?", userID).Scan(&username, &email, &avatar, &role)
	if err != nil {
		jsonErr(w, "user not found", http.StatusNotFound)
		return
	}
	jsonOK(w, map[string]any{
		"id":       userID,
		"username": username,
		"email":    email,
		"avatar":   avatar,
		"role":     role,
	})
}

// UpdateProfile updates username/email. POST /api/user/profile
func UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonErr(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	userID, _, ok := ParseToken(r)
	if !ok {
		jsonErr(w, "login required", http.StatusUnauthorized)
		return
	}

	var body struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "invalid body", http.StatusBadRequest)
		return
	}

	body.Username = strings.TrimSpace(body.Username)
	body.Email = strings.TrimSpace(body.Email)
	if body.Username == "" || body.Email == "" {
		jsonErr(w, "username and email required", http.StatusBadRequest)
		return
	}
	if len(body.Username) < 3 || len(body.Username) > 50 {
		jsonErr(w, "username must be 3-50 characters", http.StatusBadRequest)
		return
	}

	_, err := db.Conn.Exec("UPDATE users SET username = ?, email = ? WHERE id = ?", body.Username, body.Email, userID)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			jsonErr(w, "username or email already taken", http.StatusConflict)
			return
		}
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok"})
}

// ── Avatar upload ────────────────────────────────────

// Allowed image magic bytes
var imageMagic = map[string][]byte{
	"jpg":  {0xFF, 0xD8, 0xFF},
	"png":  {0x89, 0x50, 0x4E, 0x47},
	"webp": {0x52, 0x49, 0x46, 0x46}, // RIFF header
	"gif":  {0x47, 0x49, 0x46, 0x38}, // GIF8 header
}

// Blocked extensions
var blockedExts = map[string]bool{
	".php": true, ".phtml": true, ".php5": true, ".php7": true,
	".svg": true, ".htm": true, ".html": true, ".js": true,
	".jsp": true, ".asp": true, ".aspx": true, ".cgi": true,
	".sh": true, ".bat": true, ".exe": true, ".py": true,
}

const maxAvatarSize = 3 * 1024 * 1024 // 3MB

// UploadAvatar handles avatar upload. POST /api/user/avatar
func UploadAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonErr(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	userID, _, ok := ParseToken(r)
	if !ok {
		jsonErr(w, "login required", http.StatusUnauthorized)
		return
	}

	// Parse multipart (max 2MB + overhead)
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize+512)
	if err := r.ParseMultipartForm(maxAvatarSize + 512); err != nil {
		jsonErr(w, "file too large (max 3MB)", http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		jsonErr(w, "avatar file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > maxAvatarSize {
		jsonErr(w, "file too large (max 3MB)", http.StatusRequestEntityTooLarge)
		return
	}

	// Check extension is not blocked
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if blockedExts[ext] {
		jsonErr(w, "file type not allowed", http.StatusBadRequest)
		return
	}

	// Read first 16 bytes for magic byte check
	buf := make([]byte, 16)
	n, err := file.Read(buf)
	if err != nil || n < 3 {
		jsonErr(w, "cannot read file", http.StatusBadRequest)
		return
	}

	// Validate magic bytes
	detectedType := ""
	for typ, magic := range imageMagic {
		if n >= len(magic) {
			match := true
			for i, b := range magic {
				if buf[i] != b {
					match = false
					break
				}
			}
			if match {
				detectedType = typ
				break
			}
		}
	}
	if detectedType == "" {
		jsonErr(w, "invalid image file (only JPG, PNG, WEBP, GIF allowed)", http.StatusBadRequest)
		return
	}

	// Seek back to start
	file.Seek(0, io.SeekStart)

	// Generate random UUID filename
	uuidBytes := make([]byte, 16)
	rand.Read(uuidBytes)
	uuid := hex.EncodeToString(uuidBytes)
	filename := fmt.Sprintf("%s.%s", uuid, detectedType)

	// Ensure uploads directory exists
	uploadDir := "uploads/avatars"
	os.MkdirAll(uploadDir, 0755)

	// Write file
	outPath := filepath.Join(uploadDir, filename)
	out, err := os.Create(outPath)
	if err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		os.Remove(outPath)
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Delete old avatar file if exists
	var oldAvatar string
	db.Conn.QueryRow("SELECT COALESCE(avatar,'') FROM users WHERE id = ?", userID).Scan(&oldAvatar)
	if oldAvatar != "" {
		oldPath := strings.TrimPrefix(oldAvatar, "/")
		os.Remove(oldPath)
	}

	// Update database
	avatarURL := "/api/" + uploadDir + "/" + filename
	db.Conn.Exec("UPDATE users SET avatar = ? WHERE id = ?", avatarURL, userID)

	jsonOK(w, map[string]string{"avatar": avatarURL})
}
