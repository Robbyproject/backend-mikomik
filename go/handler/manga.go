package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings" // <-- Ditambahkan untuk parsing URL

	"mikomik-backend/model"
	"mikomik-backend/provider"
)

func MangaList(w http.ResponseWriter, r *http.Request) {
	// Izinkan CORS agar Frontend bisa memanggil API
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	if page <= 0 {
		page = 1
	}

	// Tangkap parameter dari Frontend
	sortParam := query.Get("sort")
	periodParam := query.Get("period")
	tagParam := query.Get("tag")
	listParam := query.Get("list")

	mangaDex := provider.NewMangaDexProvider()

	var data []model.UniversalManga
	var err error

	// Arahkan ke fungsi MangaDex yang tepat berdasarkan parameter 'sort'
	if sortParam == "latest" {
		data, err = mangaDex.GetLatestChapters(r.Context(), page)
	} else {
		data, err = mangaDex.GetCustom(r.Context(), page, sortParam, periodParam, tagParam, listParam)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	// Kirim JSON response ke Frontend
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": data,
	})
}

// =========================================================================
// ENDPOINT AKTIF UNTUK DETAIL DAN CHAPTER LIST
// =========================================================================

func MangaDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ambil ID dari URL (contoh: /api/manga/detail/12345)
	path := strings.TrimPrefix(r.URL.Path, "/api/manga/detail/")
	mangaID := strings.TrimRight(path, "/")

	if mangaID == "" {
		http.Error(w, `{"error": "manga_id required"}`, http.StatusBadRequest)
		return
	}

	mangaDex := provider.NewMangaDexProvider()
	detail, err := mangaDex.GetMangaDetail(r.Context(), mangaID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"data": detail})
}

func ChapterList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ambil ID dari URL (contoh: /api/chapter/12345/list)
	path := strings.TrimPrefix(r.URL.Path, "/api/chapter/")
	mangaID := strings.TrimSuffix(path, "/list")

	if mangaID == "" {
		http.Error(w, `{"error": "manga_id required"}`, http.StatusBadRequest)
		return
	}

	mangaDex := provider.NewMangaDexProvider()
	chapters, err := mangaDex.GetMangaChapters(r.Context(), mangaID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"data": chapters})
}

func ChapterDetail(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    // 👇 UBAH BAGIAN INI: Ganti /read/ menjadi /detail/
    chapterID := strings.TrimPrefix(r.URL.Path, "/api/chapter/detail/")
    chapterID = strings.TrimSpace(chapterID)

    if chapterID == "" {
        http.Error(w, `{"error": "chapter_id is missing"}`, http.StatusBadRequest)
        return
    }

    mangaDex := provider.NewMangaDexProvider()
    images, err := mangaDex.GetChapterImages(r.Context(), chapterID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]interface{}{"data": images})
}

func SansekaiProxyList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.Error(w, `{"error": "Sansekai proxy is permanently disabled"}`, http.StatusNotImplemented)
}