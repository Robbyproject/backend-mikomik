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

	"mikomik-backend/aggregator"
	"mikomik-backend/model"
	"mikomik-backend/provider"
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

	// Jika request spesifik minta MangaDex, cegat di sini!
	if query.Get("provider") == "mangadex" {
		page, _ := strconv.Atoi(query.Get("page"))
		if page == 0 {
			page = 1
		}

		// Siapkan HANYA provider MangaDex
		mangaDex := provider.NewMangaDexProvider()
		providers := []provider.MangaProvider{mangaDex}

		// Gunakan mesin Aggregator untuk memproses data dari MangaDex saja
		data, err := aggregator.FetchPopularManga(page, providers)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": data,
		})
		return
	}

	// FITUR AGGREGATOR LAMA: Jika ini request "Paling Populer" (sort=view)
	if query.Get("sort") == "view" {
		page, _ := strconv.Atoi(query.Get("page"))
		if page == 0 {
			page = 1
		}

		shinigami := provider.NewShinigamiProvider(scraperAPIKey, upstreamBase)
		mangaDex := provider.NewMangaDexProvider()

		providers := []provider.MangaProvider{shinigami, mangaDex}

		data, err := aggregator.FetchPopularManga(page, providers)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": data,
		})
		return
	}

	// JIKA BUKAN MANGADEX & BUKAN POPULER: Gunakan cara lama (Proxy ScraperAPI ke Shinigami)
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

// Handler untuk mengambil data Sansekai dari Cache
func SansekaiProxyList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Baca parameter "?type=latest" atau "?type=popular"
	category := r.URL.Query().Get("type")
	var cache *model.SansekaiCache

	if category == "popular" {
		cache = model.PopularCache
	} else {
		cache = model.LatestCache // Default ke latest
	}

	// Gunakan RLock untuk membaca data dengan aman (Thread-Safe)
	cache.RLock()
	data := cache.Data
	cache.RUnlock()

	// Jika data belum ada (saat server baru di-restart)
	if data == nil {
		http.Error(w, `{"error": "Data sedang disiapkan dari server, coba beberapa detik lagi"}`, http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(data)
}