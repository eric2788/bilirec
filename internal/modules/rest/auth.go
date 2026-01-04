package rest

import (
	"crypto/subtle"
	"time"

	"github.com/eric2788/bilirec/internal/modules/config"
	"github.com/eric2788/bilirec/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	User string `json:"user" form:"user"`
	Pass string `json:"pass" form:"pass"`
}

// Login
// @Summary Login and get JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body loginRequest true "Login credentials"
// @Success 200 {string} string "Login successful; JWT token is set in cookie"
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Unauthorized"
// @Router /login [post]
func loginHandler(cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {

		var req loginRequest
		if err := c.Bind().All(&req); err != nil {
			return fiber.ErrBadRequest
		}

		user := req.User
		pass := req.Pass

		if !compareUsernameAndPassword(cfg, user, pass) {
			return fiber.ErrUnauthorized
		}

		expire := time.Now().Add(time.Hour * 72)

		claims := jwt.MapClaims{
			"name": cfg.Username,
			"iat":  time.Now().Unix(),
			"exp":  expire.Unix(),
			"iss":  "bilirec",
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		t, err := token.SignedString([]byte(cfg.JwtSecret))
		if err != nil {
			return fiber.ErrInternalServerError
		}

		c.Cookie(&fiber.Cookie{
			Expires:  expire,
			Name:     jwtTokenKey,
			Value:    t,
			Path:     "/",
			Domain:   utils.Ternary(cfg.ProductionMode, cfg.BackendHost, ""),
			HTTPOnly: cfg.ProductionMode,
			Secure:   cfg.ProductionMode,
			SameSite: "None",
		})

		return c.SendStatus(fiber.StatusOK)
	}
}

func compareUsernameAndPassword(cfg *config.Config, user, pass string) bool {
	return subtle.ConstantTimeCompare([]byte(user), []byte(cfg.Username)) == 1 || bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(pass)) == nil
}
