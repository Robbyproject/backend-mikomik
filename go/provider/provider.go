package provider

import (
	"context"
	"mikomik-backend/model" // <-- UBAH INI
)

// MangaProvider adalah kontrak wajib untuk semua sumber komik
type MangaProvider interface {
	GetName() string
	GetPopular(ctx context.Context, page int) ([]model.UniversalManga, error)
	// Nanti kita bisa tambah: GetLatest(ctx, page), SearchManga(ctx, query), dll
}