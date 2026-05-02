package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"mikomik-backend/model"
)

type MangaDexProvider struct {
	Client *http.Client
}

func NewMangaDexProvider() *MangaDexProvider {
	return &MangaDexProvider{
		Client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (m *MangaDexProvider) GetName() string {
	return "MangaDex"
}

// 1. GetPopular: Mengambil manga terpopuler
func (m *MangaDexProvider) GetPopular(ctx context.Context, page int) ([]model.UniversalManga, error) {
	limit := 30
	offset := (page - 1) * limit
	targetURL := fmt.Sprintf("https://api.mangadex.org/manga?includes[]=cover_art&order[followedCount]=desc&limit=%d&offset=%d&contentRating[]=safe&contentRating[]=suggestive", limit, offset)

	return m.fetchFromMangaDex(ctx, targetURL)
}

// 2. GetCustom: Menangani berbagai filter (Rating, Seasonal, Tag, dll)
func (m *MangaDexProvider) GetCustom(ctx context.Context, page int, sort, period, tag, listID string) ([]model.UniversalManga, error) {
	limit := 30
	offset := (page - 1) * limit

	baseURL := "https://api.mangadex.org/manga"
	params := url.Values{}
	params.Add("includes[]", "cover_art")
	params.Add("limit", strconv.Itoa(limit))
	params.Add("offset", strconv.Itoa(offset))
	params.Add("contentRating[]", "safe")
	params.Add("contentRating[]", "suggestive")

	switch sort {
	case "rating":
		params.Add("order[rating]", "desc")
	case "recently_added":
		params.Add("order[createdAt]", "desc")
	case "popular_new":
		params.Add("order[followedCount]", "desc")
		oneMonthAgo := time.Now().AddDate(0, -1, 0).Format("2006-01-02T15:04:05")
		params.Add("createdAtSince", oneMonthAgo)
	default:
		params.Add("order[followedCount]", "desc")
	}

	if period != "" {
		now := time.Now()
		var since string
		if period == "harian" {
			since = now.AddDate(0, 0, -1).Format("2006-01-02T15:04:05")
		} else if period == "mingguan" {
			since = now.AddDate(0, 0, -7).Format("2006-01-02T15:04:05")
		}
		if since != "" {
			params.Add("createdAtSince", since)
		}
	}

	if tag == "doujinshi" {
		params.Add("includedTags[]", "8987b7a6-2c5e-49b4-93e5-0219c6769151")
	}

	if listID == "seasonal_spring_2026" {
		params.Add("year", "2026")
	}

	targetURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	return m.fetchFromMangaDex(ctx, targetURL)
}

// 3. GetLatestChapters: Logika 2 Tahap agar Real-time & Ada Gambar
func (m *MangaDexProvider) GetLatestChapters(ctx context.Context, page int) ([]model.UniversalManga, error) {
	limit := 30
	offset := (page - 1) * limit

	// TAHAP 1: Ambil list chapter terbaru
	chURL := fmt.Sprintf("https://api.mangadex.org/chapter?limit=%d&offset=%d&order[readableAt]=desc&includes[]=manga&contentRating[]=safe&contentRating[]=suggestive", limit, offset)

	req, _ := http.NewRequestWithContext(ctx, "GET", chURL, nil)
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chResp struct {
		Data []struct {
			Attributes struct {
				Chapter    string    `json:"chapter"`
				ReadableAt time.Time `json:"readableAt"`
			} `json:"attributes"`
			Relationships []struct {
				Type string `json:"type"`
				ID   string `json:"id"`
			} `json:"relationships"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chResp); err != nil {
		return nil, err
	}

	// Mapping ID Manga dan Info Chapter
	var mangaIDs []string
	chapterMap := make(map[string]string)
	timeMap := make(map[string]time.Time)

	for _, ch := range chResp.Data {
		for _, rel := range ch.Relationships {
			if rel.Type == "manga" {
				mangaIDs = append(mangaIDs, rel.ID)
				chapterMap[rel.ID] = ch.Attributes.Chapter
				timeMap[rel.ID] = ch.Attributes.ReadableAt
			}
		}
	}

	if len(mangaIDs) == 0 {
		return []model.UniversalManga{}, nil
	}

	// TAHAP 2: Ambil Detail Manga (Cover & Judul) berdasarkan ID dari Tahap 1
	mangaParams := url.Values{}
	for _, id := range mangaIDs {
		mangaParams.Add("ids[]", id)
	}
	mangaParams.Add("includes[]", "cover_art")
	mangaParams.Add("limit", strconv.Itoa(limit))

	finalURL := fmt.Sprintf("https://api.mangadex.org/manga?%s", mangaParams.Encode())

	return m.fetchFromMangaDexWithExtra(ctx, finalURL, chapterMap, timeMap)
}

// Fungsi Fetch Utama
func (m *MangaDexProvider) fetchFromMangaDex(ctx context.Context, targetURL string) ([]model.UniversalManga, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mdResp mdResponse
	if err := json.NewDecoder(resp.Body).Decode(&mdResp); err != nil {
		return nil, err
	}

	return m.mapToUniversal(mdResp, nil, nil), nil
}

// Fungsi Fetch Khusus dengan info Chapter & Waktu
func (m *MangaDexProvider) fetchFromMangaDexWithExtra(ctx context.Context, targetURL string, cMap map[string]string, tMap map[string]time.Time) ([]model.UniversalManga, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mdResp mdResponse
	if err := json.NewDecoder(resp.Body).Decode(&mdResp); err != nil {
		return nil, err
	}

	return m.mapToUniversal(mdResp, cMap, tMap), nil
}

// Struct internal untuk decoding MangaDex
type mdResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Attributes struct {
			Title struct {
				En string `json:"en"`
				Ja string `json:"ja-ro"`
			} `json:"title"`
			Status           string `json:"status"`
			OriginalLanguage string `json:"originalLanguage"`
		} `json:"attributes"`
		Relationships []struct {
			Type       string `json:"type"`
			Attributes struct {
				FileName string `json:"fileName"`
			} `json:"attributes"`
		} `json:"relationships"`
	} `json:"data"`
}

// Helper untuk mengubah data MangaDex ke format Universal aplikasi kita
func (m *MangaDexProvider) mapToUniversal(md mdResponse, cMap map[string]string, tMap map[string]time.Time) []model.UniversalManga {
	var list []model.UniversalManga
	for _, item := range md.Data {
		title := item.Attributes.Title.En
		if title == "" {
			title = item.Attributes.Title.Ja
		}

		fileName := ""
		for _, rel := range item.Relationships {
			if rel.Type == "cover_art" {
				fileName = rel.Attributes.FileName
			}
		}

		coverURL := ""
		if fileName != "" {
			coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.256.jpg", item.ID, fileName)
		}

		// Hitung waktu relatif jika ada
		statusStr := item.Attributes.Status
		descStr := ""
		if tMap != nil {
			diff := time.Since(tMap[item.ID])
			if diff.Hours() < 1 {
				statusStr = fmt.Sprintf("%d mnt", int(diff.Minutes()))
			} else if diff.Hours() < 24 {
				statusStr = fmt.Sprintf("%d jam", int(diff.Hours()))
			} else {
				statusStr = fmt.Sprintf("%d hari", int(diff.Hours()/24))
			}
		}
		if cMap != nil {
			descStr = "Ch. " + cMap[item.ID]
		}

		list = append(list, model.UniversalManga{
			ID:            item.ID,
			Title:         title,
			CoverImageURL: coverURL,
			Description:   descStr,
			Status:        statusStr,
			Source:        "MangaDex",
		})
	}
	return list
}

// FUNGSI BARU 1: Mengambil Detail Komik
func (m *MangaDexProvider) GetMangaDetail(ctx context.Context, id string) (*model.MangaDetailInfo, error) {
	targetURL := fmt.Sprintf("https://api.mangadex.org/manga/%s?includes[]=author&includes[]=artist&includes[]=cover_art", id)

	req, _ := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mdResp struct {
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Title       map[string]string `json:"title"`
				Description map[string]string `json:"description"`
				Status      string            `json:"status"`
				Year        int               `json:"year"`
				Tags        []struct {
					Attributes struct {
						Name map[string]string `json:"name"`
					} `json:"attributes"`
				} `json:"tags"`
			} `json:"attributes"`
			Relationships []struct {
				Type       string `json:"type"`
				Attributes struct {
					Name     string `json:"name"`
					FileName string `json:"fileName"`
				} `json:"attributes"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mdResp); err != nil {
		return nil, err
	}

	data := mdResp.Data
	
	// Ambil Judul & Deskripsi (prioritas Inggris)
	title := data.Attributes.Title["en"]
	if title == "" {
		for _, t := range data.Attributes.Title { title = t; break }
	}
	
	desc := data.Attributes.Description["en"]
	if desc == "" {
		desc = data.Attributes.Description["id"]
	}

	// Ambil Cover, Author, dan Genre
	coverURL := ""
	author := "Unknown"
	var genres []string

	for _, rel := range data.Relationships {
		if rel.Type == "cover_art" {
			coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", data.ID, rel.Attributes.FileName)
		} else if rel.Type == "author" {
			author = rel.Attributes.Name
		}
	}

	for _, tag := range data.Attributes.Tags {
		if name, ok := tag.Attributes.Name["en"]; ok {
			genres = append(genres, name)
		}
	}

	return &model.MangaDetailInfo{
		ID:            data.ID,
		Title:         title,
		Description:   desc,
		CoverImageURL: coverURL,
		Author:        author,
		Status:        data.Attributes.Status,
		Genres:        genres,
		Year:          data.Attributes.Year,
	}, nil
}

// FUNGSI BARU 2: Mengambil Daftar Chapter
func (m *MangaDexProvider) GetMangaChapters(ctx context.Context, id string) ([]model.ChapterItem, error) {
	// Limit 100 chapter terbaru. Bahasa ID & EN.
	targetURL := fmt.Sprintf("https://api.mangadex.org/manga/%s/feed?limit=100&order[chapter]=desc&translatedLanguage[]=id&translatedLanguage[]=en", id)

	req, _ := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chResp struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Chapter            string    `json:"chapter"`
				Title              string    `json:"title"`
				TranslatedLanguage string    `json:"translatedLanguage"`
				PublishAt          time.Time `json:"publishAt"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chResp); err != nil {
		return nil, err
	}

	var chapters []model.ChapterItem
	for _, ch := range chResp.Data {
		chapters = append(chapters, model.ChapterItem{
			ID:        ch.ID,
			Chapter:   ch.Attributes.Chapter,
			Title:     ch.Attributes.Title,
			Language:  ch.Attributes.TranslatedLanguage,
			CreatedAt: ch.Attributes.PublishAt.Format("02 Jan 2006"),
		})
	}
	return chapters, nil
}

// FUNGSI BARU 3: Mengambil Gambar Lembaran Komik (Chapter Detail)
func (m *MangaDexProvider) GetChapterImages(ctx context.Context, chapterID string) ([]string, error) {
	// Endpoint khusus MangaDex "at-home/server" untuk mendapatkan URL gambar komik
	targetURL := fmt.Sprintf("https://api.mangadex.org/at-home/server/%s", chapterID)

	req, _ := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var mdResp struct {
		BaseUrl string `json:"baseUrl"`
		Chapter struct {
			Hash string   `json:"hash"`
			Data []string `json:"data"` // Berisi daftar nama file gambar kualitas tinggi
		} `json:"chapter"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mdResp); err != nil {
		return nil, err
	}

	var imageUrls []string
	// Susun URL lengkap untuk masing-masing gambar
	for _, filename := range mdResp.Chapter.Data {
		// Format MangaDex: {baseUrl}/data/{hash}/{filename}
		imgURL := fmt.Sprintf("%s/data/%s/%s", mdResp.BaseUrl, mdResp.Chapter.Hash, filename)
		imageUrls = append(imageUrls, imgURL)
	}

	return imageUrls, nil
}