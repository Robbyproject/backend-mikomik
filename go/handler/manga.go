package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const upstreamBase = "https://api.shngm.io"

// API Key yang sudah Anda tes dan berhasil
const scraperAPIKey = "d45d3542c5d2af84f1b5da3d5b05ffb1"

var client = &http.Client{
	Timeout: 40 * time.Second, // Beri waktu lebih lama untuk proses proxy
}

func MangaList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Buat URL upstream, pastikan tidak ada double '?' jika query kosong
	rawQuery := r.URL.RawQuery
	upstream := fmt.Sprintf("%s/v1/manga/list", upstreamBase)
	if rawQuery != "" {
		upstream = fmt.Sprintf("%s?%s", upstream, rawQuery)
	}

	proxyGet(w, r, upstream)
}

func MangaDetail(w http.ResponseWriter, r *http.Request) {
	mangaID := extractPathParam(r.URL.Path, "/api/manga/detail/")
	if mangaID == "" {
		http.Error(w, "manga_id required", http.StatusBadRequest)
		return
	}
	upstream := fmt.Sprintf("%s/v1/manga/detail/%s", upstreamBase, mangaID)
	proxyGet(w, r, upstream)
}

func ChapterList(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api/chapter/")
	mangaID := strings.TrimSuffix(path, "/list")
	upstream := fmt.Sprintf("%s/v1/chapter/%s/list", upstreamBase, mangaID)
	if r.URL.RawQuery != "" {
		upstream = fmt.Sprintf("%s?%s", upstream, r.URL.RawQuery)
	}
	proxyGet(w, r, upstream)
}

func ChapterDetail(w http.ResponseWriter, r *http.Request) {
	chapterID := extractPathParam(r.URL.Path, "/api/chapter/detail/")
	upstream := fmt.Sprintf("%s/v1/chapter/detail/%s", upstreamBase, chapterID)
	proxyGet(w, r, upstream)
}

func proxyGet(w http.ResponseWriter, r *http.Request, upstream string) {
	// PENTING: Gunakan url.QueryEscape agar URL target tidak berantakan saat dikirim ke ScraperAPI
	encodedURL := url.QueryEscape(upstream)

	// Format URL persis seperti yang Anda tes di browser
	scraperURL := fmt.Sprintf("http://api.scraperapi.com?api_key=%s&url=%s", scraperAPIKey, encodedURL)

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, scraperURL, nil)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Gagal menghubungi ScraperAPI", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy header penting
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*") // Pastikan CORS tetap jalan
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func extractPathParam(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	return strings.TrimRight(s, "/")
}
