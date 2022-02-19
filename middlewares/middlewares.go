package middlewares

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func Auth(redis *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, _ := c.Cookie("token")
		redisPayload, err := ValidateToken(token, redis)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			c.Abort()
			return
		}
		c.Request.Header.Set("payload", redisPayload)
		c.Next()
	}
}
func ValidateToken(authorizationHeader string, redis *redis.Client) (string, error) {
	if !strings.Contains(authorizationHeader, "Bearer") {
		return "", errors.New("invalid-token")
	}
	tokenString := strings.Replace(authorizationHeader, "Bearer ", "", -1)

	redisPayload, err := redis.Get(context.Background(), tokenString).Result()
	if err != nil {
		return "", err
	}

	if redisPayload == "" {
		return "", errors.New("empty-payload")
	}

	return redisPayload, nil
}
