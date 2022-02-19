package controllers

import (
	"budgetingapi/models"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"gotest.tools/assert"
)

func TestAuthenticate(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	assert.Equal(t, nil, err)

	api := NewAPI()
	api.Db = db

	// nil request (400)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var genericResp GenericResponse

	req, _ := http.NewRequest("POST", "", nil)
	c.Request = req
	api.Authenticate(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid request", genericResp.Message)

	// bad request (400)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload := parsePayload(models.AuthRequest{})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Authenticate(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-email-or-password", genericResp.Message)

	// err select (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.AuthRequest{
		Email:    "test@gmail.com",
		Password: "test1234",
	})

	dbMock.ExpectQuery("SELECT id.*").WillReturnError(errors.New("err-select"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Authenticate(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// invalid email (401)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.AuthRequest{
		Email:    "test@gmail.com",
		Password: "test1234",
	})

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "role", "created_at", "updated_at", "is_correct"}))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Authenticate(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "invalid-email-or-password", genericResp.Message)

	// invalid password (401)
	reqAuth := models.AuthRequest{
		Email:    "test@gmail.com",
		Password: "test1234",
	}
	mockUUID := "d234578a-ee95-4dab-b5ed-e0a83b03bbfc"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqAuth)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "role", "created_at", "updated_at", "is_correct"}).
			AddRow(mockUUID, "test@gmail.com", "test", models.Admin, time.Now(), time.Now(), false))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Authenticate(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "invalid-email-or-password", genericResp.Message)

	// err generate token (500)
	redisDB, redisMock := redismock.NewClientMock()
	api.Redis = redisDB

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqAuth)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "role", "created_at", "updated_at", "is_correct"}).
			AddRow(mockUUID, "test@gmail.com", "test", models.Admin, time.Now(), time.Now(), true))

	redisMock.ExpectGet("auth:" + reqAuth.Email).SetVal("test")
	redisMock.Regexp().ExpectSet("[.]", "[.]", 30*time.Minute).SetErr(errors.New("err-set"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Authenticate(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-set", genericResp.Message)

	// err generate token (200)
	redisDB, redisMock = redismock.NewClientMock()
	api.Redis = redisDB

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqAuth)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "role", "created_at", "updated_at", "is_correct"}).
			AddRow(mockUUID, "test@gmail.com", "test", models.Admin, time.Now(), time.Now(), true))

	redisMock.ExpectGet("auth:" + reqAuth.Email).SetVal("test")
	redisMock.Regexp().ExpectSet("[.]", "[.]", 30*time.Minute).SetVal("OK")
	redisMock.Regexp().ExpectSet("[.]", "[.]", 30*time.Minute).SetVal("OK")
	redisMock.Regexp().ExpectSet("[.]", "[.]", 30*time.Minute).SetVal("OK")

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Authenticate(c)

	var respOK models.AuthResponse

	err = json.NewDecoder(w.Body).Decode(&respOK)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, reqAuth.Email, respOK.Email)
	assert.Equal(t, "test", respOK.Name)
}

func TestCheckSession(t *testing.T) {
	api := NewAPI()

	redisDB, redisMock := redismock.NewClientMock()
	api.Redis = redisDB

	// err redis (500)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var genericResp GenericResponse

	redisMock.ExpectGet("auth:test@gmail.com").SetErr(errors.New("err-redis"))

	req, _ := http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"email\":\"test@gmail.com\"}}")
	api.CheckSession(c)

	err := json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-redis", genericResp.Message)

	// unauthorized (401)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet("auth:test@gmail.com").SetErr(redis.Nil)

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"email\":\"test@gmail.com\"}}")
	api.CheckSession(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "unauthorized", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet("auth:test@gmail.com").SetVal("OK")

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"email\":\"test@gmail.com\"}}")
	api.CheckSession(c)

	genericRespOk := struct {
		Message string `json:"message"`
	}{}

	err = json.NewDecoder(w.Body).Decode(&genericRespOk)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", genericRespOk.Message)
}

func TestRefreshSession(t *testing.T) {
	api := NewAPI()

	redisDB, redisMock := redismock.NewClientMock()
	api.Redis = redisDB

	// err redis refresh token (500)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var genericResp GenericResponse

	redisMock.ExpectGet("test-refresh").SetErr(errors.New("err-redis"))

	req, _ := http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.RefreshSession(c)

	err := json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-redis", genericResp.Message)

	// unauthorized refresh token (401)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet("test-refresh").SetErr(redis.Nil)

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.RefreshSession(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "unauthorized", genericResp.Message)

	// invalid refresh payload (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet("test-refresh").SetVal("")

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.RefreshSession(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "unexpected end of JSON input", genericResp.Message)

	// err redis auth (500)
	authResponseByte, _ := json.Marshal(models.AuthResponse{
		Token: "test-token", User: models.User{Email: "test@gmail.com"},
	})
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet("test-refresh").SetVal(string(authResponseByte))
	redisMock.ExpectGet("auth:test@gmail.com").SetErr(errors.New("err-redis-auth"))

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.RefreshSession(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-redis-auth", genericResp.Message)

	// unauthorized redis auth (401)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet("test-refresh").SetVal(string(authResponseByte))
	redisMock.ExpectGet("auth:test@gmail.com").SetErr(redis.Nil)

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.RefreshSession(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "unauthorized", genericResp.Message)

	// err generate token (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet("test-refresh").SetVal(string(authResponseByte))
	redisMock.ExpectGet("auth:test@gmail.com").SetVal("")
	redisMock.Regexp().ExpectSet("[.]", "[.]", 30*time.Minute).SetErr(errors.New("err-set-generate-token"))

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.RefreshSession(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-set-generate-token", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet("test-refresh").SetVal(string(authResponseByte))
	redisMock.ExpectGet("auth:test@gmail.com").SetVal("")
	redisMock.Regexp().ExpectSet("[.]", "[.]", 30*time.Minute).SetVal("OK")
	redisMock.Regexp().ExpectSet("[.]", "[.]", 30*time.Minute).SetVal("OK")
	redisMock.Regexp().ExpectSet("[.]", "[.]", 30*time.Minute).SetVal("OK")

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.RefreshSession(c)

	var respOk models.AuthResponse

	err = json.NewDecoder(w.Body).Decode(&respOk)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test@gmail.com", respOk.User.Email)
}

func TestLogout(t *testing.T) {
	api := NewAPI()

	redisDB, redisMock := redismock.NewClientMock()
	api.Redis = redisDB

	// err redis token string (500)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var genericResp GenericResponse

	redisMock.ExpectDel("").SetErr(errors.New("err-redis-token-string"))

	req, _ := http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.Logout(c)

	err := json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-redis-token-string", genericResp.Message)

	// err redis refresh token (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectDel("").SetVal(1)
	redisMock.ExpectDel("test-refresh").SetErr(errors.New("err-redis-refresh-token"))

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.Logout(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-redis-refresh-token", genericResp.Message)

	// err redis auth email (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectDel("").SetVal(1)
	redisMock.ExpectDel("test-refresh").SetVal(1)
	redisMock.ExpectDel("auth:test@gmail.com").SetErr(errors.New("err-redis-auth-email"))

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.Logout(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-redis-auth-email", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectDel("").SetVal(1)
	redisMock.ExpectDel("test-refresh").SetVal(1)
	redisMock.ExpectDel("auth:test@gmail.com").SetVal(1)

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"refresh-token\":\"test-refresh\",\"user\":{\"email\":\"test@gmail.com\"}}")
	api.Logout(c)

	genericRespOk := struct {
		Message string `json:"message"`
	}{}

	err = json.NewDecoder(w.Body).Decode(&genericRespOk)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", genericRespOk.Message)
}
