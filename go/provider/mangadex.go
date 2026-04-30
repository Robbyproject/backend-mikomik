package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"mikomik-backend/model" // <-- UBAH INI
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

func (m *MangaDexProvider) GetPopular(ctx context.Context, page int) ([]model.UniversalManga, error) {
	// Offset hitungan dari page. (Page 1 = 0, Page 2 = 30)
	limit := 30
	offset := (page - 1) * limit

	// Endpoint MangaDex untuk komik berurut berdasarkan Follower terbanyak
	targetURL := fmt.Sprintf("https://api.mangadex.org/manga?includes[]=cover_art&order[followedCount]=desc&limit=%d&offset=%d", limit, offset)

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

	// Bentuk JSON MangaDex sedikit berbeda, jadi kita parsing manual secara sederhana
	var mdResp struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Title struct {
					En string `json:"en"`
					Ja string `json:"ja-ro"`
				} `json:"title"`
				Status string `json:"status"`
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
		
		// Ambil judul (Coba Inggris dulu, kalau kosong ambil Romaji Jepang)
		title := item.Attributes.Title.En
		if title == "" {
			title = item.Attributes.Title.Ja
		}

		// Cari URL Cover Art dari relasi data MangaDex
		coverURL := ""
		for _, rel := range item.Relationships {
			if rel.Type == "cover_art" {
				// Format URL gambar MangaDex
				coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s", item.ID, rel.Attributes.FileName)
				break
			}
		}

		universalList = append(universalList, model.UniversalManga{
			ID:            item.ID,
			Title:         title,
			CoverImageURL: coverURL,
			Status:        item.Attributes.Status,
			Rating:        0, // MangaDex pakai API terpisah untuk rating
			ViewCount:     0,
			Source:        "MangaDex",
		})
	}

	return universalList, nil
}