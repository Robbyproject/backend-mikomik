package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"mikomik-backend/db"
)

type statsResp struct {
	TotalUsers     int `json:"total_users"`
	NewToday       int `json:"new_today"`
	AdminCount     int `json:"admin_count"`
	TotalComments  int `json:"total_comments"`
	TotalBookmarks int `json:"total_bookmarks"`
}

// AdminStats returns aggregate statistics (admin only).
func AdminStats(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	_, role, ok := ParseToken(r)
	if !ok || role != "admin" {
		jsonErr(w, "forbidden", http.StatusForbidden)
		return
	}

	var s statsResp
	db.Conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&s.TotalUsers)
	db.Conn.QueryRow("SELECT COUNT(*) FROM users WHERE DATE(created_at) = CURDATE()").Scan(&s.NewToday)
	db.Conn.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&s.AdminCount)
	db.Conn.QueryRow("SELECT COUNT(*) FROM comments").Scan(&s.TotalComments)
	db.Conn.QueryRow("SELECT COUNT(*) FROM bookmarks").Scan(&s.TotalBookmarks)

	jsonOK(w, s)
}

// AdminUsers returns a paginated list of users (admin only). 10 per page.
func AdminUsers(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	_, role, ok := ParseToken(r)
	if !ok || role != "admin" {
		jsonErr(w, "forbidden", http.StatusForbidden)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 10
	offset := (page - 1) * perPage
	search := r.URL.Query().Get("search")

	var rows *sql.Rows
	var err error
	var total int

	if search != "" {
		like := "%" + search + "%"
		rows, err = db.Conn.Query(
			`SELECT id, username, email, COALESCE(avatar,''), role, created_at
			 FROM users WHERE username LIKE ? ORDER BY id DESC LIMIT ? OFFSET ?`,
			like, perPage, offset,
		)
		if err != nil {
			jsonErr(w, "internal error", http.StatusInternalServerError)
			return
		}
		db.Conn.QueryRow("SELECT COUNT(*) FROM users WHERE username LIKE ?", like).Scan(&total)
	} else {
		rows, err = db.Conn.Query(
			`SELECT id, username, email, COALESCE(avatar,''), role, created_at
			 FROM users ORDER BY id DESC LIMIT ? OFFSET ?`,
			perPage, offset,
		)
		if err != nil {
			jsonErr(w, "internal error", http.StatusInternalServerError)
			return
		}
		db.Conn.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
	}
	defer rows.Close()

	type adminUser struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		Avatar    string `json:"avatar"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	}

	users := []adminUser{}
	for rows.Next() {
		var u adminUser
		var t time.Time
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Avatar, &u.Role, &t); err == nil {
			u.CreatedAt = t.Format(time.RFC3339)
			users = append(users, u)
		}
	}

	jsonOK(w, map[string]any{
		"users":    users,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

// AdminUserDetail returns detailed info for a single user (admin only).
// GET /api/admin/user/detail?id=xxx
func AdminUserDetail(w http.ResponseWriter, r *http.Request) {
	if db.Conn == nil {
		jsonErr(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	_, role, ok := ParseToken(r)
	if !ok || role != "admin" {
		jsonErr(w, "forbidden", http.StatusForbidden)
		return
	}

	targetID, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if targetID < 1 {
		jsonErr(w, "id required", http.StatusBadRequest)
		return
	}

	// User info
	var username, email, avatar, userRole string
	var createdAt time.Time
	err := db.Conn.QueryRow(
		"SELECT username, email, COALESCE(avatar,''), role, created_at FROM users WHERE id = ?", targetID,
	).Scan(&username, &email, &avatar, &userRole, &createdAt)
	if err != nil {
		jsonErr(w, "user not found", http.StatusNotFound)
		return
	}

	// Comment count
	var commentCount int
	db.Conn.QueryRow("SELECT COUNT(*) FROM comments WHERE user_id = ?", targetID).Scan(&commentCount)

	// Recent comments
	type userComment struct {
		ID        int64  `json:"id"`
		ChapterID string `json:"chapter_id"`
		Text      string `json:"text"`
		CreatedAt string `json:"created_at"`
	}
	commentRows, _ := db.Conn.Query(
		"SELECT id, chapter_id, text, created_at FROM comments WHERE user_id = ? ORDER BY created_at DESC LIMIT 100",
		targetID,
	)
	comments := []userComment{}
	if commentRows != nil {
		defer commentRows.Close()
		for commentRows.Next() {
			var c userComment
			var t time.Time
			if commentRows.Scan(&c.ID, &c.ChapterID, &c.Text, &t) == nil {
				c.CreatedAt = t.Format(time.RFC3339)
				comments = append(comments, c)
			}
		}
	}

	// Bookmark count
	var bookmarkCount int
	db.Conn.QueryRow("SELECT COUNT(*) FROM bookmarks WHERE user_id = ?", targetID).Scan(&bookmarkCount)

	// History count
	var historyCount int
	db.Conn.QueryRow("SELECT COUNT(DISTINCT manga_id) FROM reading_history WHERE user_id = ?", targetID).Scan(&historyCount)

	// Reaction count
	var reactionCount int
	db.Conn.QueryRow("SELECT COUNT(*) FROM reactions WHERE user_id = ?", targetID).Scan(&reactionCount)

	jsonOK(w, map[string]any{
		"user": map[string]any{
			"id":         targetID,
			"username":   username,
			"email":      email,
			"avatar":     avatar,
			"role":       userRole,
			"created_at": createdAt.Format(time.RFC3339),
		},
		"stats": map[string]any{
			"comments":  commentCount,
			"bookmarks": bookmarkCount,
			"history":   historyCount,
			"reactions": reactionCount,
		},
		"recent_comments": comments,
	})
}
