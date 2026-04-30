package aggregator

import (
	"context"
	"fmt"
	"time"
	"mikomik-backend/model"    // <-- UBAH INI
	"mikomik-backend/provider" // <-- UBAH INI
)

// FetchPopularManga mencoba mengambil data dari banyak provider secara berurutan
func FetchPopularManga(page int, providers []provider.MangaProvider) ([]model.UniversalManga, error) {
	
	for _, p := range providers {
		fmt.Printf("🔄 Mencoba mengambil data populer dari [%s]...\n", p.GetName())

		// Batas waktu! Jika lebih dari 5 detik, anggap Shinigami/Provider gagal
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		resultCh := make(chan []model.UniversalManga, 1)
		errCh := make(chan error, 1)

		// Jalankan fungsi fetch di latar belakang
		go func(prov provider.MangaProvider, currentCtx context.Context) {
			data, err := prov.GetPopular(currentCtx, page)
			if err != nil {
				errCh <- err
				return
			}
			resultCh <- data
		}(p, ctx)

		select {
		case <-ctx.Done():
			fmt.Printf("⚠️ [%s] Terlalu lama (Timeout 5 detik)! Pindah gigi...\n", p.GetName())
			cancel() // Batalkan request HTTP yang nyangkut
			continue // Lanjut ke MangaDex/Provider selanjutnya

		case err := <-errCh:
			fmt.Printf("❌ [%s] Error: %v. Pindah gigi...\n", p.GetName(), err)
			cancel()
			continue

		case data := <-resultCh:
			fmt.Printf("✅ Berhasil mendapatkan data dari [%s]!\n", p.GetName())
			cancel()
			return data, nil // Langsung selesai jika berhasil!
		}
	}

	return nil, fmt.Errorf("semua sumber komik sedang down atau diblokir Cloudflare")
}