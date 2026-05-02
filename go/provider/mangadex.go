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
	return &MangaDexProvider{Client: &http.Client{}}
}

func (m *MangaDexProvider) GetName() string {
	return "MangaDex"
}

// 1. FUNGSI ORIGINAL (Jangan diubah agar Aggregator Shinigami tidak error)
func (m *MangaDexProvider) GetPopular(ctx context.Context, page int) ([]model.UniversalManga, error) {
	limit := 30
	offset := (page - 1) * limit
	targetURL := fmt.Sprintf("https://api.mangadex.org/manga?includes[]=cover_art&order[followedCount]=desc&limit=%d&offset=%d", limit, offset)

	return m.fetchFromMangaDex(ctx, targetURL)
}

// 2. FUNGSI BARU: GetCustom (Untuk menangani Popular, Recommended, Seasonal, dll)
func (m *MangaDexProvider) GetCustom(ctx context.Context, page int, sort, period, tag, listID string) ([]model.UniversalManga, error) {
	limit := 30
	offset := (page - 1) * limit

	baseURL := "https://api.mangadex.org/manga"
	params := url.Values{}
	params.Add("includes[]", "cover_art")
	params.Add("limit", strconv.Itoa(limit))
	params.Add("offset", strconv.Itoa(offset))

	// Logika Sorting
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

	// Logika Period (Harian / Mingguan)
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
			params.Set("order[followedCount]", "desc") 
		}
	}

	// Logika Tag (Self-Published / Doujinshi)
	if tag == "doujinshi" || tag == "self-published" {
		params.Add("includedTags[]", "8987b7a6-2c5e-49b4-93e5-0219c6769151")
	}

	// Logika List/Seasonal
	if listID == "seasonal_spring_2026" {
		params.Add("year", "2026")
	}

	targetURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	return m.fetchFromMangaDex(ctx, targetURL)
}

// 3. FUNGSI BARU: GetLatestChapters (Khusus untuk Latest Update agar dapat real-time chapter)
func (m *MangaDexProvider) GetLatestChapters(ctx context.Context, page int) ([]model.UniversalManga, error) {
	limit := 30
	offset := (page - 1) * limit
	
	targetURL := fmt.Sprintf("https://api.mangadex.org/chapter?includes[]=manga&includes[]=cover_art&order[readableAt]=desc&limit=%d&offset=%d&contentRating[]=safe&contentRating[]=suggestive", limit, offset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

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
				Type       string `json:"type"`
				ID         string `json:"id"`
				Attributes struct {
					Title struct {
						En string `json:"en"`
					} `json:"title"`
					FileName string `json:"fileName"`
				} `json:"attributes"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chResp); err != nil {
		return nil, err
	}

	var universalList []model.UniversalManga
	for _, ch := range chResp.Data {
		var mangaID, mangaTitle, fileName string
		for _, rel := range ch.Relationships {
			if rel.Type == "manga" {
				mangaID = rel.ID
				mangaTitle = rel.Attributes.Title.En
			}
			if rel.Type == "cover_art" {
				fileName = rel.Attributes.FileName
			}
		}

		// Hitung waktu relatif
		timeDiff := time.Since(ch.Attributes.ReadableAt)
		timeStr := ""
		if timeDiff.Hours() < 1 {
			timeStr = fmt.Sprintf("%d mnt", int(timeDiff.Minutes()))
		} else if timeDiff.Hours() < 24 {
			timeStr = fmt.Sprintf("%d jam", int(timeDiff.Hours()))
		} else {
			timeStr = fmt.Sprintf("%d hari", int(timeDiff.Hours()/24))
		}

		chapterStr := ch.Attributes.Chapter
		if chapterStr == "" {
			chapterStr = "Oneshot"
		}

		universalList = append(universalList, model.UniversalManga{
			ID:            mangaID,
			Title:         mangaTitle,
			CoverImageURL: fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.256.jpg", mangaID, fileName),
			Source:        "MangaDex",
			Description:   fmt.Sprintf("Ch. %s", chapterStr), // Simpan nomor chapter di sini
			Status:        timeStr,                          // Simpan waktu relatif di sini
		})
	}
	return universalList, nil
}

// Fungsi Bantuan Fetch
func (m *MangaDexProvider) fetchFromMangaDex(ctx context.Context, targetURL string) ([]model.UniversalManga, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mangadex API returned status %d", resp.StatusCode)
	}

	var mdResp struct {
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

	if err := json.NewDecoder(resp.Body).Decode(&mdResp); err != nil {
		return nil, err
	}

	var universalList []model.UniversalManga
	for _, item := range mdResp.Data {
		title := item.Attributes.Title.En
		if title == "" {
			title = item.Attributes.Title.Ja
		}

		coverURL := ""
		for _, rel := range item.Relationships {
			if rel.Type == "cover_art" {
				coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.256.jpg", item.ID, rel.Attributes.FileName)
				break
			}
		}

		country := "jp"
		if item.Attributes.OriginalLanguage == "ko" {
			country = "kr"
		} else if item.Attributes.OriginalLanguage == "zh" || item.Attributes.OriginalLanguage == "zh-hk" {
			country = "cn"
		}

		universalList = append(universalList, model.UniversalManga{
			ID:            item.ID,
			Title:         title,
			CoverImageURL: coverURL,
			Status:        item.Attributes.Status,
			Source:        "MangaDex",
			Country:       country,
		})
	}

	return universalList, nil
}