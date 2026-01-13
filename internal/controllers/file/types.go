package file

type PresignedURLResponse struct {
	URL       string `json:"url"`
	ExpiresIn int    `json:"expires_in"` // seconds
}
