package signeddownload

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const DefaultExpireAfter = 24 * time.Hour

type Client struct {
	jwtSecret []byte
}

type DownloadTokenClaims struct {
	FilePath string `json:"file"`
	Exp      int64  `json:"exp"`
	jwt.RegisteredClaims
}

func NewClient(secret []byte) *Client {
	return &Client{
		jwtSecret: secret,
	}
}

func (s *Client) GenerateDownloadToken(filePath string, exp int64) (string, error) {
	claims := DownloadTokenClaims{
		FilePath: filePath,
		Exp:      exp,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Client) ParseDownloadToken(tokenString string) (*DownloadTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &DownloadTokenClaims{}, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*DownloadTokenClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
