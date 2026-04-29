package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"mikomik-backend/db"
)

// ToggleBookmark adds or removes a bookmark. POST /api/user/bookmarks
func ToggleBookmark(w http.ResponseWriter, r *http.Request) {
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
		ContentType string `json:"content_type"`
		MangaID     string `json:"manga_id"`
		Title       string `json:"title"`
		Cover       string `json:"cover"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.MangaID == "" {
		jsonErr(w, "manga_id required", http.StatusBadRequest)
		return
	}
	if body.ContentType == "" {
		body.ContentType = "manga"
	}

	// Check if already bookmarked
	var id int64
	err := db.Conn.QueryRow(
		"SELECT id FROM bookmarks WHERE user_id = ? AND content_type = ? AND manga_id = ?",
		userID, body.ContentType, body.MangaID,
	).Scan(&id)
	if err == nil {
		// Exists → remove
		db.Conn.Exec("DELETE FROM bookmarks WHERE id = ?", id)
		jsonOK(w, map[string]any{"bookmarked": false})
		return
	}
	// Insert
	db.Conn.Exec(
		"INSERT INTO bookmarks (user_id, content_type, manga_id, title, cover) VALUES (?, ?, ?, ?, ?)",
		userID, body.ContentType, body.MangaID, body.Title, body.Cover,
	)
	jsonOK(w, map[string]any{"bookmarked": true})
}

type bookmarkEntry struct {
	ContentType string `json:"content_type"`
	MangaID     string `json:"manga_id"`
	Title       string `json:"title"`
	Cover       string `json:"cover"`
	CreatedAt   string `json:"created_at"`
}

// GetBookmarks returns user's bookmark list. GET /api/user/bookmarks
func GetBookmarks(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonOK(w, map[string]any{"bookmarks": []any{}})
		return
	}
	userID, _, ok := ParseToken(r)
	if !ok {
		jsonErr(w, "login required", http.StatusUnauthorized)
		return
	}

	rows, err := db.Conn.Query(
		"SELECT content_type, manga_id, title, cover, created_at FROM bookmarks WHERE user_id = ? ORDER BY created_at DESC LIMIT 200",
		userID,
	)
	if err != nil {
		jsonOK(w, map[string]any{"bookmarks": []any{}})
		return
	}
	defer rows.Close()

	list := []bookmarkEntry{}
	for rows.Next() {
		var b bookmarkEntry
		var t time.Time
		if err := rows.Scan(&b.ContentType, &b.MangaID, &b.Title, &b.Cover, &t); err == nil {
			b.CreatedAt = t.Format(time.RFC3339)
			list = append(list, b)
		}
	}
	jsonOK(w, map[string]any{"bookmarks": list})
}

// CheckBookmark checks if a content is bookmarked. GET /api/user/bookmarks/check?manga_id=x&content_type=manga
func CheckBookmark(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonOK(w, map[string]any{"bookmarked": false})
		return
	}
	userID, _, ok := ParseToken(r)
	if !ok {
		jsonOK(w, map[string]any{"bookmarked": false})
		return
	}
	mangaID := r.URL.Query().Get("manga_id")
	contentType := r.URL.Query().Get("content_type")
	if contentType == "" {
		contentType = "manga"
	}
	if mangaID == "" {
		jsonOK(w, map[string]any{"bookmarked": false})
		return
	}
	var id int64
	err := db.Conn.QueryRow(
		"SELECT id FROM bookmarks WHERE user_id = ? AND content_type = ? AND manga_id = ?",
		userID, contentType, mangaID,
	).Scan(&id)
	jsonOK(w, map[string]any{"bookmarked": err == nil})
}
