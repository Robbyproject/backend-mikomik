package handler

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"mikomik-backend/db"
)

// GetSettings returns public website settings. GET /api/settings
func GetSettings(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonOK(w, map[string]string{
			"site_title":       "MIKOMIK",
			"site_description": "Baca komik gratis",
			"site_keywords":    "manga, anime",
			"site_favicon":     "/favicon.svg",
		})
		return
	}

	rows, err := db.Conn.Query("SELECT key_name, value FROM settings")
	if err != nil {
		jsonErr(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			settings[k] = v
		}
	}

	jsonOK(w, settings)
}

// UpdateSettings updates website settings (Admin only). POST /api/admin/settings
func UpdateSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonErr(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	_, role, ok := ParseToken(r)
	if !ok || role != "admin" {
		jsonErr(w, "forbidden", http.StatusForbidden)
		return
	}

	var payload map[string]string
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		jsonErr(w, "invalid payload", http.StatusBadRequest)
		return
	}

	for k, v := range payload {
		db.Conn.Exec(
			"INSERT INTO settings (key_name, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value)",
			k, v,
		)
	}

	jsonOK(w, map[string]string{"status": "ok"})
}

// UploadFavicon handles favicon uploads (Admin only). POST /api/admin/settings/favicon
func UploadFavicon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonErr(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	_, role, ok := ParseToken(r)
	if !ok || role != "admin" {
		jsonErr(w, "forbidden", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		jsonErr(w, "file too large (max 1MB)", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("favicon")
	if err != nil {
		jsonErr(w, "invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read first 512 bytes to sniff content type
	headerBytes := make([]byte, 512)
	n, _ := file.Read(headerBytes)
	file.Seek(0, io.SeekStart) // Reset pointer

	// Allow standard image types + specifically check for ICO/SVG
	mimeType := http.DetectContentType(headerBytes[:n])
	isValidMime := mimeType == "image/jpeg" || mimeType == "image/png" || mimeType == "image/webp" || mimeType == "image/gif"

	// Fallback check for SVG and ICO since DetectContentType might return text/xml or application/octet-stream
	if !isValidMime {
		// Very basic check for SVG
		if bytes.Contains(headerBytes[:n], []byte("<svg")) {
			isValidMime = true
			mimeType = "image/svg+xml"
		}
		// Basic check for ICO (starts with 00 00 01 00)
		if n >= 4 && headerBytes[0] == 0x00 && headerBytes[1] == 0x00 && headerBytes[2] == 0x01 && headerBytes[3] == 0x00 {
			isValidMime = true
			mimeType = "image/x-icon"
		}
	}

	if !isValidMime {
		jsonErr(w, "only jpeg, png, webp, gif, ico, and svg allowed", http.StatusBadRequest)
		return
	}

	ext := ".jpg"
	if mimeType == "image/png" {
		ext = ".png"
	} else if mimeType == "image/webp" {
		ext = ".webp"
	} else if mimeType == "image/gif" {
		ext = ".gif"
	} else if mimeType == "image/svg+xml" {
		ext = ".svg"
	} else if mimeType == "image/x-icon" {
		ext = ".ico"
	}

	uploadDir := "./uploads/system"
	os.MkdirAll(uploadDir, 0755)

	uuidBytes := make([]byte, 16)
	rand.Read(uuidBytes)
	uuidStr := hex.EncodeToString(uuidBytes)
	fileName := uuidStr + ext
	filePath := filepath.Join(uploadDir, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		jsonErr(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	io.Copy(dst, file)

	faviconURL := "/api/uploads/system/" + fileName

	// Save to database
	db.Conn.Exec(
		"INSERT INTO settings (key_name, value) VALUES ('site_favicon', ?) ON DUPLICATE KEY UPDATE value = VALUES(value)",
		faviconURL,
	)

	jsonOK(w, map[string]string{"favicon": faviconURL})
}
