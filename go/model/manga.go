package model

import "time"

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
}