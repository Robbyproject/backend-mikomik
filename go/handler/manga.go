package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"backend-mikomik/aggregator" // Sesuaikan nama module project Anda
	"backend-mikomik/provider"   // Sesuaikan nama module project Anda
)

const upstreamBase = "https://api.shngm.io"
const scraperAPIKey = "d45d3542c5d2af84f1b5da3d5b05ffb1"

var client = &http.Client{
	Timeout: 40 * time.Second,
}

func MangaList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	
	// 🌟 FITUR BARU: Jika ini request untuk "Paling Populer" (sort=view), aktifkan Aggregator!
	if query.Get("sort") == "view" {
		page, _ := strconv.Atoi(query.Get("page"))
		if page == 0 {
			page = 1
		}

		// Siapkan Provider (Shinigami sebagai Utama, MangaDex sebagai Cadangan)
		shinigami := provider.NewShinigamiProvider(scraperAPIKey, upstreamBase)
		mangaDex := provider.NewMangaDexProvider()

		providers := []provider.MangaProvider{shinigami, mangaDex}

		// Jalankan Mesin Aggregator
		data, err := aggregator.FetchPopularManga(page, providers)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// Kirim hasil ke Frontend dalam format JSON
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": data,
		})
		return
	}

	// JIKA BUKAN POPULER: Gunakan cara lama (Proxy ScraperAPI)
	rawQuery := r.URL.RawQuery
	upstream := fmt.Sprintf("%s/v1/manga/list", upstreamBase)
	if rawQuery != "" {
		upstream = fmt.Sprintf("%s?%s", upstream, rawQuery)
	}

	proxyGet(w, r, upstream)
}

// ... (Biarkan MangaDetail, ChapterList, ChapterDetail tetap sama persis seperti sebelumnya) ...
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
	encodedURL := url.QueryEscape(upstream)
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

func extractPathParam(path, prefix string) string {
	s := strings.TrimPrefix(path, prefix)
	return strings.TrimRight(s, "/")
}