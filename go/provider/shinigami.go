package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"mikomik-backend/model" // <-- UBAH INI
)

type ShinigamiProvider struct {
	ScraperAPIKey string
	UpstreamBase  string
	Client        *http.Client
}

func NewShinigamiProvider(apiKey, upstream string) *ShinigamiProvider {
	return &ShinigamiProvider{
		ScraperAPIKey: apiKey,
		UpstreamBase:  upstream,
		Client:        &http.Client{},
	}
}

func (s *ShinigamiProvider) GetName() string {
	return "Shinigami"
}

func (s *ShinigamiProvider) GetPopular(ctx context.Context, page int) ([]model.UniversalManga, error) {
	targetURL := fmt.Sprintf("%s/v1/manga/list?sort=view&page=%d", s.UpstreamBase, page)
	encodedURL := url.QueryEscape(targetURL)
	scraperURL := fmt.Sprintf("http://api.scraperapi.com?api_key=%s&url=%s", s.ScraperAPIKey, encodedURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scraperURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("shinigami API returned status %d", resp.StatusCode)
	}

	var apiResp model.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	// Konversi dari format Shinigami ke format UniversalManga
	var universalList []model.UniversalManga
	for _, m := range apiResp.Data {
		
		statusStr := "Ongoing"
		if m.Status == 2 {
			statusStr = "Completed"
		}

		universalList = append(universalList, model.UniversalManga{
			ID:            m.MangaID,
			Title:         m.Title,
			CoverImageURL: m.CoverPortraitURL,
			Status:        statusStr,
			Rating:        m.UserRate,
			ViewCount:     m.ViewCount,
			Source:        "Shinigami",
		})
	}

	return universalList, nil
}