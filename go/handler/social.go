package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"mikomik-backend/db"
)

// ─── Comments ────────────────────────────────────────

type commentResp struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	ChapterID   string `json:"chapter_id"`
	ParentID    *int64 `json:"parent_id"`
	ReplyToUser string `json:"reply_to_user,omitempty"`
	Text        string `json:"text"`
	CreatedAt   string `json:"created_at"`
}

// GetComments returns comments for a chapter.
// GET /api/social/comments?chapter_id=xxx
func GetComments(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonOK(w, map[string]any{"comments": []any{}})
		return
	}
	chapterID := r.URL.Query().Get("chapter_id")
	if chapterID == "" {
		jsonErr(w, "chapter_id required", http.StatusBadRequest)
		return
	}

	rows, err := db.Conn.Query(
		`SELECT c.id, c.user_id, u.username, COALESCE(u.avatar,''), c.chapter_id, c.parent_id, c.text, c.created_at
		 FROM comments c JOIN users u ON c.user_id = u.id
		 WHERE c.chapter_id = ? ORDER BY c.created_at ASC LIMIT 200`, chapterID,
	)
	if err != nil {
		jsonOK(w, map[string]any{"comments": []any{}})
		return
	}
	defer rows.Close()

	list := []commentResp{}
	for rows.Next() {
		var c commentResp
		var t time.Time
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &c.UserID, &c.Username, &c.Avatar, &c.ChapterID, &parentID, &c.Text, &t); err == nil {
			c.CreatedAt = t.Format(time.RFC3339)
			if parentID.Valid {
				c.ParentID = &parentID.Int64
			}
			list = append(list, c)
		}
	}

	// Resolve reply_to_user names
	userMap := make(map[int64]string)
	for _, c := range list {
		userMap[c.ID] = c.Username
	}
	for i, c := range list {
		if c.ParentID != nil {
			if name, ok := userMap[*c.ParentID]; ok {
				list[i].ReplyToUser = name
			} else {
				// Parent from a different query batch — look up
				var name string
				err := db.Conn.QueryRow(
					"SELECT u.username FROM comments c JOIN users u ON c.user_id = u.id WHERE c.id = ?", *c.ParentID,
				).Scan(&name)
				if err == nil {
					list[i].ReplyToUser = name
				}
			}
		}
	}

	jsonOK(w, map[string]any{"comments": list})
}

// PostComment adds a comment (auth required).
// POST /api/social/comments { "chapter_id": "...", "text": "...", "parent_id": null }
func PostComment(w http.ResponseWriter, r *http.Request) {
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
		ChapterID string `json:"chapter_id"`
		Text      string `json:"text"`
		ParentID  *int64 `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "invalid body", http.StatusBadRequest)
		return
	}
	body.Text = strings.TrimSpace(body.Text)
	if body.ChapterID == "" || body.Text == "" {
		jsonErr(w, "chapter_id and text required", http.StatusBadRequest)
		return
	}

	// Get username + avatar
	var username, avatar string
	db.Conn.QueryRow("SELECT username, COALESCE(avatar,'') FROM users WHERE id = ?", userID).Scan(&username, &avatar)

	var replyToUser string
	if body.ParentID != nil {
		db.Conn.QueryRow(
			"SELECT u.username FROM comments c JOIN users u ON c.user_id = u.id WHERE c.id = ?", *body.ParentID,
		).Scan(&replyToUser)
	}

	res, err := db.Conn.Exec(
		"INSERT INTO comments (user_id, chapter_id, parent_id, text) VALUES (?, ?, ?, ?)",
		userID, body.ChapterID, body.ParentID, body.Text,
	)
	if err != nil {
		jsonErr(w, "internal error", http.StatusInternalServerError)
		return
	}

	id, _ := res.LastInsertId()
	jsonOK(w, map[string]any{
		"comment": commentResp{
			ID:          id,
			UserID:      userID,
			Username:    username,
			Avatar:      avatar,
			ChapterID:   body.ChapterID,
			ParentID:    body.ParentID,
			ReplyToUser: replyToUser,
			Text:        body.Text,
			CreatedAt:   time.Now().Format(time.RFC3339),
		},
	})
}

// ─── Reactions ───────────────────────────────────────

type reactionSummary struct {
	Like int `json:"like"`
	Love int `json:"love"`
	Haha int `json:"haha"`
	Wow  int `json:"wow"`
}

type reactionResp struct {
	Summary  reactionSummary `json:"summary"`
	UserKind string          `json:"user_kind"`
}

// GetReactions returns reaction counts + current user's reaction.
func GetReactions(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonOK(w, reactionResp{})
		return
	}
	chapterID := r.URL.Query().Get("chapter_id")
	if chapterID == "" {
		jsonErr(w, "chapter_id required", http.StatusBadRequest)
		return
	}

	var s reactionSummary
	db.Conn.QueryRow("SELECT COUNT(*) FROM reactions WHERE chapter_id = ? AND kind = 'like'", chapterID).Scan(&s.Like)
	db.Conn.QueryRow("SELECT COUNT(*) FROM reactions WHERE chapter_id = ? AND kind = 'love'", chapterID).Scan(&s.Love)
	db.Conn.QueryRow("SELECT COUNT(*) FROM reactions WHERE chapter_id = ? AND kind = 'haha'", chapterID).Scan(&s.Haha)
	db.Conn.QueryRow("SELECT COUNT(*) FROM reactions WHERE chapter_id = ? AND kind = 'wow'", chapterID).Scan(&s.Wow)

	resp := reactionResp{Summary: s}

	userID, _, ok := ParseToken(r)
	if ok {
		var kind string
		err := db.Conn.QueryRow("SELECT kind FROM reactions WHERE user_id = ? AND chapter_id = ?", userID, chapterID).Scan(&kind)
		if err == nil {
			resp.UserKind = kind
		}
	}

	jsonOK(w, resp)
}

// ToggleReaction sets or removes a user's reaction.
func ToggleReaction(w http.ResponseWriter, r *http.Request) {
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
		ChapterID string `json:"chapter_id"`
		Kind      string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonErr(w, "invalid body", http.StatusBadRequest)
		return
	}
	validKinds := map[string]bool{"like": true, "love": true, "haha": true, "wow": true}
	if !validKinds[body.Kind] || body.ChapterID == "" {
		jsonErr(w, "invalid kind or chapter_id", http.StatusBadRequest)
		return
	}

	var existing string
	err := db.Conn.QueryRow("SELECT kind FROM reactions WHERE user_id = ? AND chapter_id = ?", userID, body.ChapterID).Scan(&existing)

	if err == nil {
		if existing == body.Kind {
			db.Conn.Exec("DELETE FROM reactions WHERE user_id = ? AND chapter_id = ?", userID, body.ChapterID)
			jsonOK(w, map[string]string{"action": "removed"})
			return
		}
		db.Conn.Exec("UPDATE reactions SET kind = ? WHERE user_id = ? AND chapter_id = ?", body.Kind, userID, body.ChapterID)
		jsonOK(w, map[string]string{"action": "updated"})
		return
	}

	db.Conn.Exec("INSERT INTO reactions (user_id, chapter_id, kind) VALUES (?, ?, ?)", userID, body.ChapterID, body.Kind)
	jsonOK(w, map[string]string{"action": "added"})
}
