package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"mikomik-backend/db"
)

// RecordHistory records a chapter/episode read. POST /api/user/history
func RecordHistory(w http.ResponseWriter, r *http.Request) {
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
		ContentType   string  `json:"content_type"`
		MangaID       string  `json:"manga_id"`
		ChapterID     string  `json:"chapter_id"`
		ChapterNumber float64 `json:"chapter_number"`
		Title         string  `json:"title"`
		Cover         string  `json:"cover"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.MangaID == "" || body.ChapterID == "" {
		jsonErr(w, "manga_id and chapter_id required", http.StatusBadRequest)
		return
	}
	if body.ContentType == "" {
		body.ContentType = "manga"
	}

	_, err := db.Conn.Exec(
		`INSERT INTO reading_history (user_id, content_type, manga_id, chapter_id, chapter_number, title, cover)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE chapter_number = VALUES(chapter_number), title = VALUES(title), cover = VALUES(cover), read_at = NOW()`,
		userID, body.ContentType, body.MangaID, body.ChapterID, body.ChapterNumber, body.Title, body.Cover,
	)
	if err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]string{"status": "ok"})
}

type historyEntry struct {
	ContentType   string  `json:"content_type"`
	MangaID       string  `json:"manga_id"`
	ChapterID     string  `json:"chapter_id"`
	ChapterNumber float64 `json:"chapter_number"`
	Title         string  `json:"title"`
	Cover         string  `json:"cover"`
	ReadAt        string  `json:"read_at"`
}

// GetHistory returns user's reading/watch history. GET /api/user/history
func GetHistory(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonOK(w, map[string]any{"history": []any{}})
		return
	}
	userID, _, ok := ParseToken(r)
	if !ok {
		jsonErr(w, "login required", http.StatusUnauthorized)
		return
	}

	rows, err := db.Conn.Query(
		`SELECT content_type, manga_id, chapter_id, chapter_number, title, cover, read_at
		 FROM reading_history
		 WHERE user_id = ?
		 ORDER BY read_at DESC LIMIT 200`, userID,
	)
	if err != nil {
		jsonOK(w, map[string]any{"history": []any{}})
		return
	}
	defer rows.Close()

	list := []historyEntry{}
	for rows.Next() {
		var h historyEntry
		var t time.Time
		if err := rows.Scan(&h.ContentType, &h.MangaID, &h.ChapterID, &h.ChapterNumber, &h.Title, &h.Cover, &t); err == nil {
			h.ReadAt = t.Format(time.RFC3339)
			list = append(list, h)
		}
	}
	jsonOK(w, map[string]any{"history": list})
}

// GetMangaHistory returns user's last read chapter for a specific manga/anime.
// GET /api/user/history/manga?manga_id=xxx
func GetMangaHistory(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonOK(w, map[string]any{"last_chapter": 0})
		return
	}
	userID, _, ok := ParseToken(r)
	if !ok {
		jsonOK(w, map[string]any{"last_chapter": 0})
		return
	}
	mangaID := r.URL.Query().Get("manga_id")
	if mangaID == "" {
		jsonOK(w, map[string]any{"last_chapter": 0})
		return
	}

	var chapterNumber float64
	err := db.Conn.QueryRow(
		`SELECT chapter_number FROM reading_history
		 WHERE user_id = ? AND manga_id = ?
		 ORDER BY read_at DESC LIMIT 1`, userID, mangaID,
	).Scan(&chapterNumber)
	if err != nil {
		jsonOK(w, map[string]any{"last_chapter": 0})
		return
	}
	jsonOK(w, map[string]any{"last_chapter": chapterNumber})
}
