package rest

import (
	"crypto/subtle"
	"time"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	User string `json:"user" form:"user"`
	Pass string `json:"pass" form:"pass"`
}

func loginHandler(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req loginRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.ErrBadRequest
		}

		user := req.User
		pass := req.Pass

		if subtle.ConstantTimeCompare([]byte(user), []byte(cfg.Username)) != 1 || bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(pass)) != nil {
			return fiber.ErrUnauthorized
		}

		claims := jwt.MapClaims{
			"name": cfg.Username,
			"iat":  time.Now().Unix(),
			"exp":  time.Now().Add(time.Hour * 72).Unix(),
			"iss":  "bilirec",
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		t, err := token.SignedString([]byte(cfg.JwtSecret))
		if err != nil {
			return fiber.ErrInternalServerError
		}

		return c.JSON(fiber.Map{"token": t})
	}
}
