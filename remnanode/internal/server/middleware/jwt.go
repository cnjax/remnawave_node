package middleware

import (
	"crypto/rsa"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/remnawave/remnanode/internal/errors"
)

// JWTMiddleware creates a JWT authentication middleware using RS256
func JWTMiddleware(publicKeyPEM string) gin.HandlerFunc {
	// Parse the public key once at initialization
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
	if err != nil {
		panic("failed to parse JWT public key: " + err.Error())
	}

	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			errors.SendError(c, errors.ErrUnauthorized)
			c.Abort()
			return
		}

		// Check for Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			errors.SendError(c, errors.ErrUnauthorized)
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token — require exactly RS256, reject RS384/RS512/PS*.
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
				return nil, jwt.ErrSignatureInvalid
			}
			return publicKey, nil
		})

		if err != nil || !token.Valid {
			errors.SendError(c, errors.ErrUnauthorized)
			c.Abort()
			return
		}

		// Store claims in context for later use
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("jwt_claims", claims)
		}

		c.Next()
	}
}

// GetJWTClaims retrieves JWT claims from the context
func GetJWTClaims(c *gin.Context) jwt.MapClaims {
	claims, exists := c.Get("jwt_claims")
	if !exists {
		return nil
	}
	return claims.(jwt.MapClaims)
}

// ValidateJWTPublicKey checks if the public key can be parsed
func ValidateJWTPublicKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	return jwt.ParseRSAPublicKeyFromPEM([]byte(publicKeyPEM))
}
