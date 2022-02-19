package controllers

import (
	"budgetingapi/models"
	"bytes"
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

func TestGetProducts(t *testing.T) {
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

	dbMock.ExpectQuery("SELECT p.id.*").WillReturnError(fmt.Errorf("err-select"))

	req, _ := http.NewRequest("GET", "", nil)
	c.Request = req
	api.GetProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// scan error (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	label := []string{"id", "name", "description",
		"category_id", "user_id", "created_at",
		"updated_at", "category_name", "category_desription"}

	dbMock.ExpectQuery("SELECT p.id.*").
		WillReturnRows(sqlmock.NewRows(label).AddRow(mockID, "dummy", "dummy", mockUserID, mockID, false, false, "", ""))

	req, _ = http.NewRequest("GET", "?order_by=id", nil)
	c.Request = req
	api.GetProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, true, strings.Contains(genericResp.Message, "sql: Scan error"))

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT p.id.*").WithArgs(mockUserID).
		WillReturnRows(sqlmock.NewRows(label).AddRow(mockID, "dummy", "dummy", mockUserID, mockID, time.Now(), time.Now(), "", ""))
	dbMock.ExpectQuery("SELECT COUNT.*").WillReturnError(fmt.Errorf("err-count"))

	req, _ = http.NewRequest("GET", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-count", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT p.id.*").WithArgs(mockUserID, "%"+"123"+"%", "%"+"asd"+"%", mockID).
		WillReturnRows(sqlmock.NewRows(label).AddRow(mockID, "dummy", "dummy", mockUserID, mockID, time.Now(), time.Now(), "", ""))
	dbMock.ExpectQuery("SELECT COUNT.*").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	req, _ = http.NewRequest("GET", "?name=123&description=asd&category_id="+mockID, nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetProducts(c)

	var resp models.ProductList
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, int(resp.Total))
	assert.Equal(t, 1, len(resp.Products))
	assert.Equal(t, mockID, resp.Products[0].Id)
	assert.Equal(t, mockUserID, resp.Products[0].UserId)

	// as excel
	// products not found (404)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT p.id.*").WithArgs(mockUserID).
		WillReturnRows(sqlmock.NewRows(label))

	req, _ = http.NewRequest("GET", "?export_as_excel=true", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "products-not-found", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT p.id.*").WithArgs(mockUserID).
		WillReturnRows(sqlmock.NewRows(label).AddRow(mockID, "dummy", "dummy", mockUserID, mockID, time.Now(), time.Now(), "", ""))
	req, _ = http.NewRequest("GET", "?export_as_excel=true", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetProducts(c)

	loc, _ := time.LoadLocation("Asia/Jakarta")
	fileName := fmt.Sprintf("report_products_%s.xlsx", time.Now().In(loc).Format("20060102_150405"))

	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "attachment;filename=\""+fileName+"\"", w.Header()["Content-Disposition"][0])

}

func TestUpsertProducts(t *testing.T) {
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
	api.UpsertProducts(c)

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
	api.UpsertProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-products", genericResp.Message)

	// err begin (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.UpsertProductRequest{Data: []models.Product{
		{},
	}})

	dbMock.ExpectBegin().WillReturnError(fmt.Errorf("err-begin"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpsertProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-begin", genericResp.Message)

	// products validation & insert failure (500)
	respErrors := struct {
		Message string            `json:"message"`
		Details []models.RowError `json:"details"`
	}{}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	mockID := "63eb226a-d612-412b-b8d4-a3e17b7d2226"
	mockID2 := "63eb226a-d612-412b-b8d4-a3e17b7d2227"
	mockUserID := "63eb226a-d612-412b-b8d4-a3e17b7d2228"
	products := models.UpsertProductRequest{Data: []models.Product{
		{},
		{Id: mockID, Name: "product 1"},
		{Id: mockID, Name: "product 2", Description: "product 2 desc"},
		{Id: mockID, Name: "product 3", Description: "product 3 desc", UserId: "err"},
		{Id: mockID, Name: "product 4", Description: "product 4 desc", UserId: mockUserID},
		{Id: mockID, Name: "product 5", Description: "product 4 desc", UserId: mockUserID, CategoryId: mockID},
	}}
	dataOK := products.Data[5]
	payload = parsePayload(products)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO products.*").
		WithArgs(dataOK.Id, dataOK.Name, dataOK.Description, dataOK.UserId, dataOK.CategoryId).WillReturnError(fmt.Errorf("err-insert"))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"ADMIN\"}}")
	api.UpsertProducts(c)

	err = json.NewDecoder(w.Body).Decode(&respErrors)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "error", respErrors.Message)
	assert.Equal(t, 6, len(respErrors.Details))
	assert.Equal(t, "missing-name", respErrors.Details[0].Message)
	assert.Equal(t, "missing-description", respErrors.Details[1].Message)
	assert.Equal(t, "missing-user-id", respErrors.Details[2].Message)
	assert.Equal(t, "invalid-user-id", respErrors.Details[3].Message)
	assert.Equal(t, "invalid-category-id", respErrors.Details[4].Message)
	assert.Equal(t, "err-insert", respErrors.Details[5].Message)

	// err commit (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	products = models.UpsertProductRequest{Data: []models.Product{
		{Id: mockID, Name: "product new 1", Description: "product new 1 desc", CategoryId: mockID},
		{Id: mockID2, Name: "product new 2", Description: "product new 2 desc", CategoryId: mockID},
		{Id: mockID, Name: "product update 1", Description: "product update 1 desc", CategoryId: mockID},
	}}
	payload = parsePayload(products)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO products.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO products.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO products.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit().WillReturnError(fmt.Errorf("err-commit"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.UpsertProducts(c)

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
	dbMock.ExpectExec("INSERT INTO products.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO products.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO products.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit()

	payload = parsePayload(products)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.UpsertProducts(c)

	err = json.NewDecoder(w.Body).Decode(&respSuccess)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", respSuccess.Message)
	assert.Equal(t, 3, respSuccess.Total)
}

func TestDeleteProducts(t *testing.T) {
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
	api.DeleteProducts(c)

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
	api.DeleteProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-data", genericResp.Message)

	// bad request (400)

	mockID := "63eb226a-d612-412b-b8d4-a3e17b7d2226"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{mockID, "error"}})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteProducts(c)

	var rowResp models.RowResponseError

	err = json.NewDecoder(w.Body).Decode(&rowResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "error", rowResp.Message)
	assert.Equal(t, 1, len(rowResp.Detail))
	assert.Equal(t, 1, rowResp.Detail[0].Row)
	assert.Equal(t, "invalid-id", rowResp.Detail[0].Message)

	// err begin (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{mockID, mockID}})

	dbMock.ExpectBegin().WillReturnError(fmt.Errorf("err-begin"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-begin", genericResp.Message)

	// exec error (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{mockID, mockID}})

	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE products.*").WillReturnError(fmt.Errorf("err-exec"))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-exec", genericResp.Message)

	// rows affected different from request (404)
	reqData := models.BatchDeleteRequest{Data: []string{mockID, mockID}}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqData)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE products.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, fmt.Sprintf("expected-%d-updated-but-got-%d", len(reqData.Data), 1), genericResp.Message)

	// err commit (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqData)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE products.*").WillReturnResult(sqlmock.NewResult(0, 2))
	dbMock.ExpectCommit().WillReturnError(fmt.Errorf("err-commit"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteProducts(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-commit", genericResp.Message)

	// 200
	mockUserID := "63eb226a-d612-412b-b8d4-a3e17b7d2227"
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqData)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE products.*").WillReturnResult(sqlmock.NewResult(0, 2))
	dbMock.ExpectCommit()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.DeleteProducts(c)

	var respOk map[string]string

	err = json.NewDecoder(w.Body).Decode(&respOk)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", respOk["message"])
}

func parsePayload(p interface{}) *bytes.Buffer {
	data, _ := json.Marshal(p)
	return bytes.NewBuffer(data)
}
