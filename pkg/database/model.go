package database

type Model struct {
	ID        int64  `json:"id"`
	Alias     string `json:"alias"`
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}
