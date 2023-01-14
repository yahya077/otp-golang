package otp_golang

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"os"
	"strings"
)

func AuthMiddleware(c *fiber.Ctx) error {
	var authMiddlewareHandler AuthMiddlewareHandler

	c.ReqHeaderParser(&authMiddlewareHandler.Header)

	if authMiddlewareHandler.HasBearer() {
		_, e := authMiddlewareHandler.ParseToken()

		if e != nil {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		if claims, ok := authMiddlewareHandler.GetMappedClaims(); ok {
			c.Locals(LocalClaims, claims)
			return c.Next()
		}
	}

	return c.SendStatus(fiber.StatusUnauthorized)
}

type AuthMiddlewareHandler struct {
	Header HeaderBearer
	Token  *jwt.Token
	Claims jwt.MapClaims
}

func (a *AuthMiddlewareHandler) HasBearer() bool {
	return strings.Contains(a.Header.Authorization, "Bearer")
}

func (a *AuthMiddlewareHandler) GetTokenString() string {
	return strings.Replace(a.Header.Authorization, "Bearer ", "", -1)
}

func (a *AuthMiddlewareHandler) ParseToken() (*jwt.Token, error) {
	var e error
	a.Token, e = jwt.Parse(a.GetTokenString(), func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(os.Getenv("JWT_SECRET_KEY")), nil
	})
	return a.Token, e
}

func (a *AuthMiddlewareHandler) GetMappedClaims() (jwt.MapClaims, bool) {
	var ok bool
	a.Claims, ok = a.Token.Claims.(jwt.MapClaims)
	ok = a.Token.Valid
	return a.Claims, ok
}
