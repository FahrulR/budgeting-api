package controllers

import (
	"budgetingapi/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"gotest.tools/assert"
)

func TestGetCategories(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	assert.Equal(t, nil, err)

	api := NewAPI()
	api.Db = db

	var genericResp GenericResponse

	// err select (500)
	mockID := "63eb226a-d612-412b-b8d4-a3e17b7d2226"
	mockUserID := "63eb226a-d612-412b-b8d4-a3e17b7d2227"
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT id.*").WillReturnError(fmt.Errorf("err-select"))

	req, _ := http.NewRequest("GET", "", nil)
	c.Request = req
	api.GetCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// scan error (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	label := []string{"id", "name", "description",
		"user_id", "created_at", "updated_at"}

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows(label).AddRow(mockID, "dummy", "dummy", mockUserID, false, false))

	req, _ = http.NewRequest("GET", "?order_by=id", nil)
	c.Request = req
	api.GetCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, true, strings.Contains(genericResp.Message, "sql: Scan error"))

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT id.*").WithArgs(mockUserID).
		WillReturnRows(sqlmock.NewRows(label).AddRow(mockID, "dummy", "dummy", mockUserID, time.Now(), time.Now()))
	dbMock.ExpectQuery("SELECT COUNT.*").WillReturnError(fmt.Errorf("err-count"))

	req, _ = http.NewRequest("GET", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-count", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT id.*").WithArgs(mockUserID, "%"+"123"+"%", "%"+"asd"+"%").
		WillReturnRows(sqlmock.NewRows(label).AddRow(mockID, "dummy", "dummy", mockUserID, time.Now(), time.Now()))
	dbMock.ExpectQuery("SELECT COUNT.*").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	req, _ = http.NewRequest("GET", "?name=123&description=asd", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetCategories(c)

	var resp models.CategoryList
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, int(resp.Total))
	assert.Equal(t, 1, len(resp.Categories))
	assert.Equal(t, mockID, resp.Categories[0].Id)
	assert.Equal(t, mockUserID, resp.Categories[0].UserId)

}

func TestUpsertCategories(t *testing.T) {
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
	api.UpsertCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid request", genericResp.Message)

	// bad request (400)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload := parsePayload(models.UpsertProductRequest{})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpsertCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-categories", genericResp.Message)

	// err begin (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.UpsertProductRequest{Data: []models.Product{
		{},
	}})

	dbMock.ExpectBegin().WillReturnError(fmt.Errorf("err-begin"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpsertCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-begin", genericResp.Message)

	// categories validation & insert failure (500)
	respErrors := struct {
		Message string            `json:"message"`
		Details []models.RowError `json:"details"`
	}{}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	mockID := "63eb226a-d612-412b-b8d4-a3e17b7d2226"
	mockID2 := "63eb226a-d612-412b-b8d4-a3e17b7d2227"
	mockUserID := "63eb226a-d612-412b-b8d4-a3e17b7d2228"
	categories := models.UpsertCategoryRequest{Data: []models.Category{
		{},
		{Id: mockID, Name: "category 1"},
		{Id: mockID, Name: "category 2", Description: "category 2 desc"},
		{Id: mockID, Name: "category 3", Description: "category 3 desc", UserId: "err"},
		{Id: mockID, Name: "category 4", Description: "product 4 desc", UserId: mockUserID},
	}}
	dataOK := categories.Data[4]
	payload = parsePayload(categories)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO categories.*").
		WithArgs(dataOK.Id, dataOK.Name, dataOK.Description, dataOK.UserId).WillReturnError(fmt.Errorf("err-insert"))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"ADMIN\"}}")
	api.UpsertCategories(c)

	err = json.NewDecoder(w.Body).Decode(&respErrors)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "error", respErrors.Message)
	assert.Equal(t, 5, len(respErrors.Details))
	assert.Equal(t, "missing-name", respErrors.Details[0].Message)
	assert.Equal(t, "missing-description", respErrors.Details[1].Message)
	assert.Equal(t, "missing-user-id", respErrors.Details[2].Message)
	assert.Equal(t, "invalid-user-id", respErrors.Details[3].Message)
	assert.Equal(t, "err-insert", respErrors.Details[4].Message)

	// err commit (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	categories = models.UpsertCategoryRequest{Data: []models.Category{
		{Id: mockID, Name: "category new 1", Description: "category new 1 desc"},
		{Id: mockID2, Name: "category new 2", Description: "category new 2 desc"},
		{Id: mockID, Name: "category update 1", Description: "category update 1 desc"},
	}}
	payload = parsePayload(categories)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO categories.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO categories.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO categories.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit().WillReturnError(fmt.Errorf("err-commit"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.UpsertCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-commit", genericResp.Message)

	// 200
	respSuccess := struct {
		Message string `json:"message"`
		Total   int    `json:"total"`
	}{}

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO categories.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO categories.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO categories.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit()

	payload = parsePayload(categories)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.UpsertCategories(c)

	err = json.NewDecoder(w.Body).Decode(&respSuccess)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", respSuccess.Message)
	assert.Equal(t, 3, respSuccess.Total)
}

func TestDeleteCategories(t *testing.T) {
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
	api.DeleteCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid request", genericResp.Message)

	// bad request (400)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload := parsePayload(models.BatchDeleteRequest{})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-data", genericResp.Message)

	// err check exist (500)
	mockID := "63eb226a-d612-412b-b8d4-a3e17b7d2226"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{mockID}})

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnError(fmt.Errorf("err-exists"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-exists", genericResp.Message)

	// bad request (400)
	mockID = "63eb226a-d612-412b-b8d4-a3e17b7d2226"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{"error", mockID}})

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteCategories(c)

	var rowResp models.RowResponseError

	err = json.NewDecoder(w.Body).Decode(&rowResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 2, len(rowResp.Detail))
	assert.Equal(t, 0, rowResp.Detail[0].Row)
	assert.Equal(t, "invalid-id", rowResp.Detail[0].Message)
	assert.Equal(t, 1, rowResp.Detail[1].Row)
	assert.Equal(t, "conflict-id", rowResp.Detail[1].Message)

	// err begin (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{mockID, mockID}})

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectBegin().WillReturnError(fmt.Errorf("err-begin"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-begin", genericResp.Message)

	// exec error (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{mockID, mockID}})

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE categories.*").WillReturnError(fmt.Errorf("err-exec"))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-exec", genericResp.Message)

	// rows affected different from request (404)
	reqData := models.BatchDeleteRequest{Data: []string{mockID, mockID}}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqData)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE categories.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, fmt.Sprintf("expected-%d-updated-but-got-%d", len(reqData.Data), 1), genericResp.Message)

	// err commit (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqData)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE categories.*").WillReturnResult(sqlmock.NewResult(0, 2))
	dbMock.ExpectCommit().WillReturnError(fmt.Errorf("err-commit"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteCategories(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-commit", genericResp.Message)

	// 200
	mockUserID := "63eb226a-d612-412b-b8d4-a3e17b7d2227"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqData)

	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectQuery("SELECT EXISTS.*").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE categories.*").WillReturnResult(sqlmock.NewResult(0, 2))
	dbMock.ExpectCommit()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.DeleteCategories(c)

	var respOk map[string]string

	err = json.NewDecoder(w.Body).Decode(&respOk)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", respOk["message"])
}
