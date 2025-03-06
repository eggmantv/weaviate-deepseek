package models

type SourceChunk struct {
	Additional struct {
		ID string `json:"id"`
	} `json:"_additional"`
	Title string `json:"title"`
	URL   string `json:"url"`
	// IconURL   string `json:"icon_url"`
	Captions  string `json:"captions"`
	MediaType string `json:"media_type"`
}
