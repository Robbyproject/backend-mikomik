package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const animeUpstream = "https://apps.animekita.org/api/v1.2.3"

var animeClient = &http.Client{
	Timeout: 15 * time.Second,
}

// AnimeList proxies GET /api/anime/list
func AnimeList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	upstream := fmt.Sprintf("%s/anime-list.php", animeUpstream)
	animeProxy(w, r, upstream)
}

// AnimeSearch proxies GET /api/anime/search?keyword=xxx&page=1&limit=20
func AnimeSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		http.Error(w, "keyword param required", http.StatusBadRequest)
		return
	}

	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "20"
	}

	upstream := fmt.Sprintf("%s/search.php?keyword=%s&page=%s&limit=%s", animeUpstream, url.QueryEscape(keyword), page, limit)
	animeProxy(w, r, upstream)
}

// AnimeSeries proxies GET /api/anime/series?url=xxx
func AnimeSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		http.Error(w, "url param required", http.StatusBadRequest)
		return
	}

	upstream := fmt.Sprintf("%s/series.php?url=%s", animeUpstream, url.QueryEscape(urlParam))
	animeProxy(w, r, upstream)
}

// AnimeEpisode proxies GET /api/anime/episode?url=xxx&reso=720p
func AnimeEpisode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	epUrl := r.URL.Query().Get("url")
	if epUrl == "" {
		http.Error(w, "url param required", http.StatusBadRequest)
		return
	}

	reso := r.URL.Query().Get("reso")
	if reso == "" {
		reso = "720p"
	}

	upstream := fmt.Sprintf("%s/chapter.php?url=%s&reso=%s", animeUpstream, url.QueryEscape(epUrl), reso)
	animeProxy(w, r, upstream)
}

func animeProxy(w http.ResponseWriter, r *http.Request, upstream string) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstream, nil)
	if err != nil {
		http.Error(w, `{"error":"failed to create request"}`, http.StatusInternalServerError)
		return
	}

	req.Header.Set("User-Agent", "Mikomik/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := animeClient.Do(req)
	if err != nil {
		http.Error(w, `{"error":"upstream request failed"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read response"}`, http.StatusInternalServerError)
		return
	}

	// Strip any leading HTML/PHP warnings before the JSON
	raw := string(body)
	jsonStart := -1
	for i, c := range raw {
		if c == '{' || c == '[' {
			jsonStart = i
			break
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")

	if jsonStart < 0 {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`{"error":"invalid upstream response"}`))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(raw[jsonStart:]))
}
