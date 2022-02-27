package controllers

import (
	"budgetingapi/models"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func (api *API) Authenticate(c *gin.Context) {
	var authRequest models.AuthRequest
	if err := c.ShouldBindJSON(&authRequest); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	if authRequest.Email == "" || authRequest.Password == "" {
		sendError(c, http.StatusBadRequest, "missing-email-or-password")
		return
	}

	var authResponse models.AuthResponse

	var correct bool
	err := api.Db.QueryRow(`
		SELECT id, email, name, role, created_at, updated_at, password = crypt($2, password)
		FROM users
		WHERE email = $1
	`, authRequest.Email, authRequest.Password).Scan(&authResponse.User.Id, &authResponse.User.Email, &authResponse.User.Name, &authResponse.User.Role,
		&authResponse.User.CreatedAt, &authResponse.User.UpdatedAt, &correct)

	if err != nil {
		if err == sql.ErrNoRows {
			sendError(c, http.StatusUnauthorized, "invalid-email-or-password")
			return
		}

		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if !correct {
		sendError(c, http.StatusUnauthorized, "invalid-email-or-password")
		return
	}

	sessPayload, _ := api.Redis.Get(context.Background(), "auth:"+authRequest.Email).Result()
	if sessPayload != "" {
		log.Println("removing old session..")
		api.Redis.Del(context.Background(), sessPayload)
	}

	authResponse.Token, err = api.GenerateToken(authResponse)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, authResponse)
}

func (api *API) CheckSession(c *gin.Context) {
	u := ParsePayload(c)

	err := api.Redis.Get(context.Background(), "auth:"+u.Email).Err()
	if err != nil {
		if err == redis.Nil {
			sendError(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, genericOK)
}

func (api *API) RefreshSession(c *gin.Context) {
	u := ParsePayload(c)

	refreshPayload, err := api.Redis.Get(context.Background(), u.RefreshToken).Result()
	if err != nil {
		if err == redis.Nil {
			sendError(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	var authResponse models.AuthResponse

	if err := json.Unmarshal([]byte(refreshPayload), &authResponse); err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	err = api.Redis.Get(context.Background(), "auth:"+u.Email).Err()
	if err != nil {
		if err == redis.Nil {
			sendError(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	authResponse.Token, err = api.GenerateToken(authResponse)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, authResponse)
}

func (api *API) Logout(c *gin.Context) {
	u := ParsePayload(c)
	token, _ := c.Cookie("token")
	tokenString := strings.Replace(token, "Bearer ", "", -1)

	err := api.Redis.Del(context.Background(), tokenString).Err()
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	err = api.Redis.Del(context.Background(), u.RefreshToken).Err()
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	err = api.Redis.Del(context.Background(), "auth:"+u.Email).Err()
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, genericOK)
}

func (api *API) ForgotPassword(c *gin.Context) {
	var authRequest models.AuthRequest
	if err := c.ShouldBindJSON(&authRequest); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	if authRequest.Email == "" {
		sendError(c, http.StatusBadRequest, "missing-email")
		return
	}

	if _, err := mail.ParseAddress(authRequest.Email); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, "invalid-email")
		return
	}

	var id string
	if err := api.Db.QueryRow("SELECT id FROM users WHERE email = $1 AND NOT deleted", authRequest.Email).Scan(&id); err != nil {
		// prevent user enumeration
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, genericOK)
			return
		}

		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	token := tokenGenerator()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sessPayload, _ := api.Redis.Get(context.Background(), "reset:"+authRequest.Email).Result()
	if sessPayload != "" {
		log.Println("removing old link password..")
		api.Redis.Del(ctx, sessPayload)
	}

	err := api.Redis.Set(ctx, "reset:"+authRequest.Email, token, 30*time.Minute).Err()
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	err = api.Redis.Set(ctx, token, id, 30*time.Minute).Err()
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := sendEmailReset(authRequest.Email, token); err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, genericOK)
}

func (api *API) VerifyTokenReset(c *gin.Context) {
	userId, err := api.CheckResetToken(c)

	// response already handled from the function
	if err != nil {
		log.Println(err)
		return
	}

	c.JSON(http.StatusOK, map[string]string{"id": userId})
}

func (api *API) UpdateUserReset(c *gin.Context) {
	userId, err := api.CheckResetToken(c)

	// response already handled from the function
	if err != nil {
		log.Println(err)
		return
	}

	var passwordResetRequest models.PasswordReset
	if err := c.ShouldBindJSON(&passwordResetRequest); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	if passwordResetRequest.Password == "" || passwordResetRequest.PasswordConfirmation == "" {
		sendError(c, http.StatusBadRequest, "missing-password-or-password-confirmation")
		return
	}

	if len(passwordResetRequest.Password) < 8 {
		sendError(c, http.StatusBadRequest, "password-at-least-8-characters")
		return
	}

	if passwordResetRequest.PasswordConfirmation != passwordResetRequest.Password {
		sendError(c, http.StatusBadRequest, "password-confirmation-does-not-match")
		return
	}

	email, err := api.UpdatePassword(userId, passwordResetRequest.Password)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	token := c.Param("token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = api.Redis.Del(ctx, "reset:"+email).Err()
	if err != nil {
		// password already changed, just print the err
		log.Println(err)
	}

	err = api.Redis.Del(ctx, token).Err()
	if err != nil {
		// password already changed, just print the err
		log.Println(err)
	}

	c.JSON(http.StatusOK, genericOK)
}

func (api *API) CheckResetToken(c *gin.Context) (userId string, err error) {
	token := c.Param("token")
	if token == "" {
		sendError(c, http.StatusBadRequest, "missing-token")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	userId, err = api.Redis.Get(ctx, token).Result()
	if err != nil {
		if err == redis.Nil {
			sendError(c, http.StatusNotFound, "token-invalid-or-expired")
			return
		}

		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
	}

	return
}

func (api *API) GenerateToken(resp models.AuthResponse) (string, error) {

	key, err := base64.StdEncoding.DecodeString(os.Getenv("SESSION_KEY"))
	if err != nil {
		log.Println(err)
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(resp.Id))
	mac.Write(key)

	sum := mac.Sum(nil)

	sEnc := base64.StdEncoding.EncodeToString(sum)
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user-id"] = resp.Id
	claims["session-id"] = sEnc
	claims["expires"] = 1800
	refreshToken, err := token.SignedString(key)
	if err != nil {
		log.Println(err)
		return "", err
	}
	claims["refresh-token"] = refreshToken
	claims["user"] = resp.User

	redisPayload, _ := json.Marshal(claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		log.Println(err)
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data := map[string]string{
		tokenString:          string(redisPayload),
		refreshToken:         string(redisPayload),
		"auth:" + resp.Email: tokenString,
	}

	for k, v := range data {
		err = api.Redis.Set(ctx, k, v, 30*time.Minute).Err()
		if err != nil {
			log.Println(err)
			return "", err
		}

	}

	auth := fmt.Sprintf("Bearer %s", tokenString)

	return auth, nil
}
