package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const upstreamBase = "https://api.shngm.io"

var client = &http.Client{
	Timeout: 15 * time.Second,
}

// MangaList proxies GET /api/manga/list to the upstream shinigami API.
func MangaList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	upstream := fmt.Sprintf("%s/v1/manga/list?%s", upstreamBase, r.URL.RawQuery)
	proxyGet(w, r, upstream)
}

// MangaDetail proxies GET /api/manga/detail/{manga_id}
func MangaDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract manga_id from path: /api/manga/detail/{manga_id}
	mangaID := extractPathParam(r.URL.Path, "/api/manga/detail/")
	if mangaID == "" {
		http.Error(w, "manga_id required", http.StatusBadRequest)
		return
	}

	upstream := fmt.Sprintf("%s/v1/manga/detail/%s", upstreamBase, mangaID)
	proxyGet(w, r, upstream)
}

// ChapterList proxies GET /api/chapter/{manga_id}/list
func ChapterList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract manga_id from path: /api/chapter/{manga_id}/list
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api/chapter/")
	mangaID := strings.TrimSuffix(path, "/list")

	if mangaID == "" {
		http.Error(w, "manga_id required", http.StatusBadRequest)
		return
	}

	upstream := fmt.Sprintf("%s/v1/chapter/%s/list?%s", upstreamBase, mangaID, r.URL.RawQuery)
	proxyGet(w, r, upstream)
}

// ChapterDetail proxies GET /api/chapter/detail/{chapter_id}
func ChapterDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	chapterID := extractPathParam(r.URL.Path, "/api/chapter/detail/")
	if chapterID == "" {
		http.Error(w, "chapter_id required", http.StatusBadRequest)
		return
	}

	upstream := fmt.Sprintf("%s/v1/chapter/detail/%s", upstreamBase, chapterID)
	proxyGet(w, r, upstream)
}

// proxyGet fetches from upstream and streams the response back.
func proxyGet(w http.ResponseWriter, r *http.Request, upstream string) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mikomik/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "upstream request failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func extractPathParam(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	// Remove trailing slash if any
	s = strings.TrimRight(s, "/")
	return s
}
