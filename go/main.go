package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"mikomik-backend/db"
	"mikomik-backend/handler"
)

func main() {
	// Initialize MariaDB (graceful fallback if unavailable)
	db.Init()

	mux := http.NewServeMux()

	// Manga endpoints
	mux.HandleFunc("/api/manga/list", handler.MangaList)
	mux.HandleFunc("/api/manga/detail/", handler.MangaDetail)
	mux.HandleFunc("/api/chapter/detail/", handler.ChapterDetail)
	mux.HandleFunc("/api/chapter/", handler.ChapterList)

	// Anime endpoints
	mux.HandleFunc("/api/anime/list", handler.AnimeList)
	mux.HandleFunc("/api/anime/search", handler.AnimeSearch)
	mux.HandleFunc("/api/anime/series", handler.AnimeSeries)
	mux.HandleFunc("/api/anime/episode", handler.AnimeEpisode)

	// Auth endpoints (rate limited)
	mux.HandleFunc("/api/auth/register", handler.RateLimit(handler.RegisterLimiter, handler.Register))
	mux.HandleFunc("/api/auth/login", handler.RateLimit(handler.LoginLimiter, handler.Login))
	mux.HandleFunc("/api/auth/me", handler.Me)
	mux.HandleFunc("/api/auth/logout", handler.Logout)

	// Admin endpoints
	mux.HandleFunc("/api/admin/stats", handler.AdminStats)
	mux.HandleFunc("/api/admin/users", handler.AdminUsers)
	mux.HandleFunc("/api/admin/user/detail", handler.AdminUserDetail)
	mux.HandleFunc("/api/admin/settings", handler.UpdateSettings)
	mux.HandleFunc("/api/admin/settings/favicon", handler.UploadFavicon)

	// Settings endpoint (Public)
	mux.HandleFunc("/api/settings", handler.GetSettings)

	// Social endpoints (comments & reactions) — no rate limit on comments
	mux.HandleFunc("/api/social/comments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.PostComment(w, r)
		} else {
			handler.GetComments(w, r)
		}
	})
	mux.HandleFunc("/api/social/reactions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.ToggleReaction(w, r)
		} else {
			handler.GetReactions(w, r)
		}
	})

	// User endpoints (history, bookmarks, profile)
	mux.HandleFunc("/api/user/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.RecordHistory(w, r)
		} else {
			handler.GetHistory(w, r)
		}
	})
	mux.HandleFunc("/api/user/history/manga", handler.GetMangaHistory)
	mux.HandleFunc("/api/user/bookmarks/check", handler.CheckBookmark)
	mux.HandleFunc("/api/user/bookmarks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.ToggleBookmark(w, r)
		} else {
			handler.GetBookmarks(w, r)
		}
	})
	mux.HandleFunc("/api/user/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handler.UpdateProfile(w, r)
		} else {
			handler.GetProfile(w, r)
		}
	})
	mux.HandleFunc("/api/user/avatar", handler.RateLimit(handler.AvatarLimiter, handler.UploadAvatar))

	// Static file servers for specific uploads (legacy)
	avatarDir := http.Dir("./uploads/avatars")
	mux.Handle("/api/uploads/avatars/", http.StripPrefix("/api/uploads/avatars/", http.FileServer(avatarDir)))

	systemDir := http.Dir("./uploads/system")
	mux.Handle("/api/uploads/system/", http.StripPrefix("/api/uploads/system/", http.FileServer(systemDir)))

	// Generic uploads handler with directory listing prevention
	// This handler should come after specific handlers to avoid shadowing them
	fs := http.StripPrefix("/api/uploads/", http.FileServer(http.Dir("uploads")))
	mux.HandleFunc("/api/uploads/", func(w http.ResponseWriter, r *http.Request) {
		// Prevent directory listing for the root /api/uploads/ path
		if r.URL.Path == "/api/uploads/" || r.URL.Path == "/api/uploads" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		fs.ServeHTTP(w, r)
	})

	// Health check
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Ambil port dari sistem (Render akan memberikan nilai ini secara otomatis)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000" // Ini adalah cadangan jika Anda menjalankan di laptop sendiri
	}

	// Wrap with CORS
	server := &http.Server{
		Addr:    ":" + port, // Gunakan variabel port di sini
		Handler: cors(mux),
	}

	fmt.Printf("🚀 MIKOMIK backend running on port %s\n", port)
	log.Fatal(server.ListenAndServe())
}

// cors middleware — allows the Vite dev server origin with credentials.
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
