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
	limit := 30
	offset := (page - 1) * limit
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

	// Tangkap "originalLanguage" untuk tahu negara asalnya
	var mdResp struct {
		Data []struct {
			ID         string `json:"id"`
			Attributes struct {
				Title struct {
					En string `json:"en"`
					Ja string `json:"ja-ro"`
				} `json:"title"`
				Status           string `json:"status"`
				OriginalLanguage string `json:"originalLanguage"` // 🌟 TANGKAP BAHASA
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
				// 🌟 FIX GAMBAR: Tambahkan .256.jpg di belakang agar ukurannya kecil dan tidak error
				coverURL = fmt.Sprintf("https://uploads.mangadex.org/covers/%s/%s.256.jpg", item.ID, rel.Attributes.FileName)
				break
			}
		}

		// 🌟 FIX NEGARA: Konversi bahasa ke kode negara
		country := "jp" // Default jepang
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
			Rating:        0,
			ViewCount:     0,
			Source:        "MangaDex",
			Country:       country, // 🌟 MASUKKAN KE SINI
		})
	}

	return universalList, nil
}