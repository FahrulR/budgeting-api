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
	"gotest.tools/assert"
)

func TestRegister(t *testing.T) {
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
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid request", genericResp.Message)

	// bad request (400)
	user := models.User{}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload := parsePayload(user)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-email", genericResp.Message)

	user.Email = "invalid"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-name", genericResp.Message)

	user.Name = "test"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid-email", genericResp.Message)

	user.Email = "test@gmail.com"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-password", genericResp.Message)

	user.Password = "test"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "password-must-be-at-least-8-characters", genericResp.Message)

	// err select exist (500)
	user.Password = "test1234"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnError(errors.New("err-select"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// email conflict (409)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "email-already-exist", genericResp.Message)

	// err insert (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectExec("INSERT INTO users.*").WillReturnError(errors.New("err-insert"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-insert", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectExec("INSERT INTO users.*").WillReturnResult(sqlmock.NewResult(0, 1))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.Register(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", genericResp.Message)
}

func TestGetUser(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	assert.Equal(t, nil, err)

	api := NewAPI()
	api.Db = db
	var genericResp GenericResponse

	mockUserID := "63eb226a-d612-412b-b8d4-a3e17b7d2227"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// missing / invalid id (400)
	req, _ := http.NewRequest("GET", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+""+"\", \"role\":\"CUSTOMER\"}}")
	api.GetUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-id", genericResp.Message)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	req, _ = http.NewRequest("GET", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+"err"+"\", \"role\":\"CUSTOMER\"}}")
	api.GetUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid-id", genericResp.Message)

	// err select (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT id.*").WillReturnError(errors.New("err-select"))

	req, _ = http.NewRequest("GET", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// not found (404)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	label := []string{"id", "email", "name", "role", "created_at", "updated_at"}
	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows(label))

	req, _ = http.NewRequest("GET", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "user-not-found", genericResp.Message)

	// not found (404)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT id.*").WithArgs(mockUserID).
		WillReturnRows(sqlmock.NewRows(label).
			AddRow(mockUserID, "test@gmail.com", "test", "CUSTOMER", time.Now(), time.Now()))

	req, _ = http.NewRequest("GET", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetUser(c)

	var user models.User

	err = json.NewDecoder(w.Body).Decode(&user)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, mockUserID, user.Id)
}

func TestUpdateUser(t *testing.T) {
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
	api.UpdateUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid request", genericResp.Message)

	// bad request (400)
	user := models.User{}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload := parsePayload(user)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-email", genericResp.Message)

	// err select exist (500)
	user.Email = "test@gmail.com"
	user.Name = "test"
	user.Password = "test1234"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnError(errors.New("err-select"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// email conflict (409)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "email-already-exist", genericResp.Message)

	// err update (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectExec("UPDATE users.*").WillReturnError(errors.New("err-update"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-update", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(user)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectExec("UPDATE users.*").WillReturnResult(sqlmock.NewResult(0, 1))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpdateUser(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", genericResp.Message)
}
