package utils

import (
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

func ToJwtClaims(c fiber.Ctx) jwt.MapClaims {
	token := jwtware.FromContext(c)
	if token == nil {
		return nil
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil
	}
	return claims
}

func GetClaimString(claims jwt.MapClaims, key string) (string, bool) {
	v, ok := claims[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}
