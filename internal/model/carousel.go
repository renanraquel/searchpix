package model

// CarouselItem mídia exibida no carrossel (imagem ou vídeo).
type CarouselItem struct {
	ID          string   `json:"id"`
	TenantID    string   `json:"tenant_id"`
	MediaType   string   `json:"media_type"` // "image" | "video"
	Title       string   `json:"title,omitempty"`
	MediaURL    string   `json:"media_url"`
	SortOrder   int      `json:"sort_order"`
	CreatedAt   FlexTime `json:"created_at"`
	UpdatedAt   FlexTime `json:"updated_at"`
}

// CarouselSettings parâmetros de exibição do carrossel por estabelecimento.
type CarouselSettings struct {
	TenantID              string `json:"tenant_id"`
	ImageDurationSeconds  int    `json:"image_duration_seconds"`
}
