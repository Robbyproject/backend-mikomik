package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// APIResponse is the top-level response from the shinigami API.
type APIResponse struct {
	Retcode int     `json:"retcode"`
	Message string  `json:"message"`
	Meta    Meta    `json:"meta"`
	Data    []Manga `json:"data"`
}

type Meta struct {
	RequestID   string `json:"request_id"`
	Timestamp   int64  `json:"timestamp"`
	ProcessTime string `json:"process_time"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
	TotalPage   int    `json:"total_page"`
	TotalRecord int    `json:"total_record"`
}

type Manga struct {
	MangaID           string                   `json:"manga_id"`
	Title             string                   `json:"title"`
	AlternativeTitle  string                   `json:"alternative_title"`
	Description       string                   `json:"description"`
	CoverImageURL     string                   `json:"cover_image_url"`
	CoverPortraitURL  string                   `json:"cover_portrait_url"`
	CountryID         string                   `json:"country_id"`
	Status            int                      `json:"status"`
	ReleaseYear       string                   `json:"release_year"`
	Rank              int                      `json:"rank"`
	UserRate          float64                  `json:"user_rate"`
	ViewCount         int                      `json:"view_count"`
	BookmarkCount     int                      `json:"bookmark_count"`
	IsRecommended     bool                     `json:"is_recommended"`
	LatestChapterID   string                   `json:"latest_chapter_id"`
	LatestChapterNum  int                      `json:"latest_chapter_number"`
	LatestChapterTime time.Time                `json:"latest_chapter_time"`
	Chapters          []Chapter                `json:"chapters"`
	Taxonomy          map[string][]TaxonomyEntry `json:"taxonomy"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
	DeletedAt         *time.Time               `json:"deleted_at"`
}

type Chapter struct {
	ChapterID     string    `json:"chapter_id"`
	ChapterNumber float64   `json:"chapter_number"`
	CreatedAt     time.Time `json:"created_at"`
}

type TaxonomyEntry struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// UniversalManga adalah bentuk standar komik agar front-end tidak kebingungan
// saat menerima data dari banyak sumber (Shinigami, MangaDex, dll)
type UniversalManga struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	CoverImageURL string  `json:"cover_image_url"`
	Status        string  `json:"status"`
	Rating        float64 `json:"user_rate"`
	ViewCount     int     `json:"view_count"`
	Source        string  `json:"source"`
	Country       string  `json:"country"`
}

// --- SISTEM CACHE SANSEKAI ---

// SansekaiCache menyimpan data dari Sansekai agar tidak perlu fetch berulang kali
type SansekaiCache struct {
	sync.RWMutex
	Data      interface{}
	UpdatedAt time.Time
}

var (
	LatestCache  = &SansekaiCache{}
	PopularCache = &SansekaiCache{}
)

func fetchFromSansekai(endpoint string) (interface{}, error) {
	url := fmt.Sprintf("https://api.sansekai.my.id/api/komik%s", endpoint)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	// Tambahkan header super lengkap menyerupai Browser Google Chrome Asli
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://api.sansekai.my.id/")
	req.Header.Set("Origin", "https://api.sansekai.my.id")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 🌟 BAGIAN DETEKTIF: Baca isi pesan error langsung dari Sansekai 🌟
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status: %d, alasan dari server: %s", resp.StatusCode, string(bodyBytes))
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

/// updateCaches
func updateCaches() {
	fmt.Println("🔄 Memperbarui Cache Sansekai...")

	// 🌟 UPDATE: Ubah 'type=all' menjadi 'type=project'
	if data, err := fetchFromSansekai("/latest?type=project&project=komikcast"); err == nil {
		LatestCache.Lock()
		LatestCache.Data = data
		LatestCache.UpdatedAt = time.Now()
		LatestCache.Unlock()
	} else {
		fmt.Println("⚠️ Gagal fetch latest dari Sansekai:", err)
	}

	time.Sleep(2 * time.Second) // Jeda agar tidak dianggap spam

	// 🌟 UPDATE: Ubah juga di sini
	if data, err := fetchFromSansekai("/popular?type=project&project=komikcast"); err == nil {
		PopularCache.Lock()
		PopularCache.Data = data
		PopularCache.UpdatedAt = time.Now()
		PopularCache.Unlock()
	} else {
		fmt.Println("⚠️ Gagal fetch popular dari Sansekai:", err)
	}
	
	fmt.Println("✅ Proses Cache Sansekai Selesai!")
}

// StartSansekaiWorker berjalan di background setiap 15 menit
func StartSansekaiWorker() {
	// Jalankan sekali di awal saat server baru menyala
	go updateCaches()

	// Jadwalkan setiap 15 menit
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		for range ticker.C {
			updateCaches()
		}
	}()
}