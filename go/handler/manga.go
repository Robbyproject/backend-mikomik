package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

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
// ENDPOINT DI BAWAH INI DINONAKTIFKAN SEMENTARA (HANYA MENGEMBALIKAN ERROR)
// Karena Shinigami dimatikan, endpoint untuk detail & baca komik 
// akan kita buat ulang khusus untuk MangaDex nanti.
// =========================================================================

func MangaDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.Error(w, `{"error": "Endpoint Detail sedang disesuaikan untuk MangaDex"}`, http.StatusNotImplemented)
}

func ChapterList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.Error(w, `{"error": "Endpoint Chapter sedang disesuaikan untuk MangaDex"}`, http.StatusNotImplemented)
}

func ChapterDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.Error(w, `{"error": "Endpoint Baca Chapter sedang disesuaikan untuk MangaDex"}`, http.StatusNotImplemented)
}

func SansekaiProxyList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.Error(w, `{"error": "Sansekai proxy is permanently disabled"}`, http.StatusNotImplemented)
}