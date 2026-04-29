package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url" // Tambahan: untuk mengenkripsi URL ke format yang aman (URL Encoding)
	"strings"
	"time"
)

const upstreamBase = "https://api.shngm.io"

// GANTI DENGAN API KEY DARI DASHBOARD SCRAPERAPI ANDA
const scraperAPIKey = "d45d3542c5d2af84f1b5da3d5b05ffb1"

var client = &http.Client{
	// Timeout dinaikkan dari 15s ke 30s karena ScraperAPI butuh waktu ekstra untuk menjebol Cloudflare
	Timeout: 30 * time.Second,
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

// proxyGet fetches from upstream and streams the response back using ScraperAPI
func proxyGet(w http.ResponseWriter, r *http.Request, upstream string) {
	// Encode URL target agar aman dimasukkan ke dalam parameter ScraperAPI
	encodedURL := url.QueryEscape(upstream)

	// Format URL ScraperAPI
	scraperURL := fmt.Sprintf("http://api.scraperapi.com?api_key=%s&url=%s&render=false", scraperAPIKey, encodedURL)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, scraperURL, nil)
	if err != nil {
		http.Error(w, "failed to create request", http.StatusInternalServerError)
		return
	}

	// ScraperAPI sudah mengurus User-Agent dan IP. Kita cukup pastikan menerima JSON.
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
	s = strings.TrimRight(s, "/")
	return s
}
