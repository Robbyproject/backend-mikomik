package model

// UniversalManga adalah bentuk standar komik untuk frontend
// Sekarang murni dioptimalkan untuk MangaDex
type UniversalManga struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	CoverImageURL string  `json:"cover_image_url"`
	Status        string  `json:"status"`
	Rating        float64 `json:"user_rate,omitempty"`
	ViewCount     int     `json:"view_count,omitempty"`
	Source        string  `json:"source"`
	Country       string  `json:"country"`
	Description   string  `json:"description,omitempty"` // Dipakai untuk menampilkan Chapter real-time
}

// StartSansekaiWorker dibiarkan sebagai fungsi kosong (dummy)
// Tujuannya agar file main.go Anda tidak error jika masih memanggil fungsi ini
func StartSansekaiWorker() {
	// Fitur Sansekai dan Shinigami resmi dimatikan.
	// Worker ini sengaja dikosongkan.
}