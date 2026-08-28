package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// genuinely fuck gin
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		//get header check not empty
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing authorization header",
			})
			c.Abort()
			return
		}

		//split and check token is proper
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header",
			})
			c.Abort()
			return
		}

		//get the token string and parse it
		tokenString := parts[1]
		token, err := jwt.Parse(
			tokenString,
			func(token *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv("JWT_SECRET")), nil
			},
		)
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
			})
			c.Abort()
			return
		}

		//get the data from token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid claims",
			})
			c.Abort()
			return
		}

		//get user id from claims
		userIDString, ok := claims["sub"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid user ID",
			})
			c.Abort()
			return
		}
		//parse the user id and set it in context
		userID, err := strconv.ParseInt(userIDString, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Couldnt Recognize the User",
			})
			c.Abort()
			return
		}
		c.Set("userID", userID)

		c.Next()
	}
}
