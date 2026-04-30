package provider

import (
	"context"
	"backend-mikomik/model" // Sesuaikan nama module go.mod Anda
)

// MangaProvider adalah kontrak wajib untuk semua sumber komik
type MangaProvider interface {
	GetName() string
	GetPopular(ctx context.Context, page int) ([]model.UniversalManga, error)
	// Nanti kita bisa tambah: GetLatest(ctx, page), SearchManga(ctx, query), dll
}