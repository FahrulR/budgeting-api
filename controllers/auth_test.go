package controllers

import (
	"budgetingapi/models"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"gopkg.in/gomail.v2"
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

	// (200)
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

func TestForgotPassword(t *testing.T) {
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
	api.ForgotPassword(c)

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
	api.ForgotPassword(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-email", genericResp.Message)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.AuthRequest{Email: "invalid"})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.ForgotPassword(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid-email", genericResp.Message)

	// err select (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.AuthRequest{
		Email: "test@gmail.com",
	})

	dbMock.ExpectQuery("SELECT id.*").WillReturnError(errors.New("err-select"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.ForgotPassword(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// user not found prevent enumeration (200)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.AuthRequest{
		Email: "test@gmail.com",
	})

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.ForgotPassword(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", genericResp.Message)

	mockUUID := "6eaa2a0c-d562-4ac6-a62a-76b7498efc7d"
	reqAuth := models.AuthRequest{
		Email: "test@gmail.com",
	}

	// err set reset redis (500)
	redisDB, redisMock := redismock.NewClientMock()
	api.Redis = redisDB

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqAuth)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(mockUUID))

	redisMock.ExpectGet("reset:" + reqAuth.Email).SetVal("test")
	redisMock.Regexp().ExpectSet("reset:"+reqAuth.Email, "^[a-z0-9]", 30*time.Minute).SetErr(errors.New("err-set-reset"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.ForgotPassword(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-set-reset", genericResp.Message)

	// err set token redis (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqAuth)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(mockUUID))

	redisMock.ExpectGet("reset:" + reqAuth.Email).SetVal("test")
	redisMock.Regexp().ExpectSet("reset:"+reqAuth.Email, "^[a-z0-9]", 30*time.Minute).SetVal("ok")
	redisMock.Regexp().ExpectSet("^[a-z0-9]", "^[a-z0-9]", 30*time.Minute).SetErr(errors.New("err-set-token"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.ForgotPassword(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-set-token", genericResp.Message)

	// err set send email invalid smtp port (500)
	oriDial := dialAndSend

	defer func() {
		dialAndSend = oriDial
	}()

	errDial := errors.New("err-dial")
	dialAndSend = func(*gomail.Dialer, ...*gomail.Message) error {
		return errDial
	}

	oriPort := os.Getenv("EMAIL_SMTP_PORT")
	os.Setenv("EMAIL_SMTP_PORT", "asd")

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqAuth)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(mockUUID))

	redisMock.ExpectGet("reset:" + reqAuth.Email).SetVal("test")
	redisMock.Regexp().ExpectSet("reset:"+reqAuth.Email, "^[a-z0-9]", 30*time.Minute).SetVal("ok")
	redisMock.Regexp().ExpectSet("^[a-z0-9]", "^[a-z0-9]", 30*time.Minute).SetVal("ok")

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.ForgotPassword(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "strconv.Atoi: parsing \"asd\": invalid syntax", genericResp.Message)

	// err dial and send (500)
	os.Setenv("EMAIL_SMTP_PORT", oriPort)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqAuth)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(mockUUID))

	redisMock.ExpectGet("reset:" + reqAuth.Email).SetVal("test")
	redisMock.Regexp().ExpectSet("reset:"+reqAuth.Email, "^[a-z0-9]", 30*time.Minute).SetVal("ok")
	redisMock.Regexp().ExpectSet("^[a-z0-9]", "^[a-z0-9]", 30*time.Minute).SetVal("ok")

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.ForgotPassword(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-dial", genericResp.Message)

	// (200)

	errDial = nil

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqAuth)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).
			AddRow(mockUUID))

	redisMock.ExpectGet("reset:" + reqAuth.Email).SetVal("test")
	redisMock.Regexp().ExpectSet("reset:"+reqAuth.Email, "^[a-z0-9]", 30*time.Minute).SetVal("ok")
	redisMock.Regexp().ExpectSet("^[a-z0-9]", "^[a-z0-9]", 30*time.Minute).SetVal("ok")

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.ForgotPassword(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", genericResp.Message)
}

func TestVerifyTokenReset(t *testing.T) {
	api := NewAPI()

	redisDB, redisMock := redismock.NewClientMock()
	api.Redis = redisDB

	// missing token (400)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var genericResp GenericResponse

	req, _ := http.NewRequest("POST", "", nil)
	c.Request = req
	api.VerifyTokenReset(c)

	err := json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-token", genericResp.Message)

	// err redis (500)
	token := tokenGenerator()
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet(token).SetErr(errors.New("err-redis"))

	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})
	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	api.VerifyTokenReset(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-redis", genericResp.Message)

	// token not found
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	redisMock.ExpectGet(token).SetErr(redis.Nil)

	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})
	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	api.VerifyTokenReset(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "token-invalid-or-expired", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	mockUUID := "8368a923-97c5-4e72-8c0c-62b961bf9d07"
	redisMock.ExpectGet(token).SetVal(mockUUID)

	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})
	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	api.VerifyTokenReset(c)

	respOk := map[string]string{}

	err = json.NewDecoder(w.Body).Decode(&respOk)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, mockUUID, respOk["id"])
}

func TestUpdateUserReset(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	assert.Equal(t, nil, err)

	redisDB, redisMock := redismock.NewClientMock()

	api := NewAPI()
	api.Db = db
	api.Redis = redisDB

	// missing token (400)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var genericResp GenericResponse

	req, _ := http.NewRequest("POST", "", nil)
	c.Request = req
	api.UpdateUserReset(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-token", genericResp.Message)

	// nil request (400)
	token := tokenGenerator()
	mockUUID := "5f32354d-699c-4459-afb7-1021bb121dad"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})

	redisMock.ExpectGet(token).SetVal(mockUUID)

	req, _ = http.NewRequest("POST", "", nil)
	c.Request = req
	api.UpdateUserReset(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid request", genericResp.Message)

	// bad request (400)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})

	redisMock.ExpectGet(token).SetVal(mockUUID)

	payload := parsePayload(models.PasswordReset{})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUserReset(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-password-or-password-confirmation", genericResp.Message)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})

	redisMock.ExpectGet(token).SetVal(mockUUID)

	payload = parsePayload(models.PasswordReset{Password: "test123", PasswordConfirmation: "test1235"})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUserReset(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "password-must-be-at-least-8-characters", genericResp.Message)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})

	redisMock.ExpectGet(token).SetVal(mockUUID)

	payload = parsePayload(models.PasswordReset{Password: "test1234", PasswordConfirmation: "test1235"})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUserReset(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "password-confirmation-does-not-match", genericResp.Message)

	// err update not found (404)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})

	redisMock.ExpectGet(token).SetVal(mockUUID)
	dbMock.ExpectQuery("UPDATE users.*").WillReturnRows(sqlmock.NewRows([]string{"email"}))

	payload = parsePayload(models.PasswordReset{Password: "test1235", PasswordConfirmation: "test1235"})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUserReset(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "not-found", genericResp.Message)

	// (200)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	c.Params = append(c.Params, gin.Param{Key: "token", Value: token})

	redisMock.ExpectGet(token).SetVal(mockUUID)
	dbMock.ExpectQuery("UPDATE users.*").WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("test@gmail.com"))
	redisMock.ExpectDel("reset:test@gmail.com").SetErr(errors.New("err-del-reset"))
	redisMock.ExpectDel(token).SetErr(errors.New("err-del-token"))

	payload = parsePayload(models.PasswordReset{Password: "test1235", PasswordConfirmation: "test1235"})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUserReset(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", genericResp.Message)

}
