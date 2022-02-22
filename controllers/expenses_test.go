package controllers

import (
	"budgetingapi/models"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"gotest.tools/assert"
)

func TestGetExpensesReport(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	assert.Equal(t, nil, err)

	api := NewAPI()
	api.Db = db

	var genericResp GenericResponse

	// err select sum (500)
	mockID := "63eb226a-d612-412b-b8d4-a3e17b7d2226"
	mockUserID := "63eb226a-d612-412b-b8d4-a3e17b7d2227"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT SUM.*").WillReturnError(errors.New("err-select-sum-idr"))

	req, _ := http.NewRequest("GET", "?category_id=3e80f025-ff3c-4b25-a7bc-883a3c432236&currency=all&min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	api.GetExpensesReport(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select-sum-idr", genericResp.Message)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT SUM.*").WillReturnError(errors.New("err-select-sum-idr"))

	req, _ = http.NewRequest("GET", "?category_id=3e80f025-ff3c-4b25-a7bc-883a3c432236&currency=IDR&min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	api.GetExpensesReport(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select-sum-idr", genericResp.Message)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT SUM.*").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(1000))
	dbMock.ExpectQuery("SELECT SUM.*").WillReturnError(errors.New("err-select-sum-usd"))

	req, _ = http.NewRequest("GET", "?category_id=3e80f025-ff3c-4b25-a7bc-883a3c432236&currency=all&min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	api.GetExpensesReport(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select-sum-usd", genericResp.Message)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT SUM.*").WillReturnError(errors.New("err-select-sum-usd"))

	req, _ = http.NewRequest("GET", "?category_id=3e80f025-ff3c-4b25-a7bc-883a3c432236&currency=USD&min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	api.GetExpensesReport(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select-sum-usd", genericResp.Message)

	// err select report (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT SUM.*").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(1000))
	dbMock.ExpectQuery("SELECT SUM.*").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(1000))
	dbMock.ExpectQuery("SELECT e.currency.*").WillReturnError(errors.New("err-select-report"))

	req, _ = http.NewRequest("GET", "?category_id=3e80f025-ff3c-4b25-a7bc-883a3c432236&currency=all&min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	api.GetExpensesReport(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select-report", genericResp.Message)

	// err scan report (500)
	label := []string{"currency", "category_id", "category_name", "amount"}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT SUM.*").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(1000))
	dbMock.ExpectQuery("SELECT SUM.*").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(1000))
	dbMock.ExpectQuery("SELECT e.currency.*").WillReturnRows(sqlmock.NewRows(label[1:]).AddRow(mockID, "test", 1234))

	req, _ = http.NewRequest("GET", "?category_id=3e80f025-ff3c-4b25-a7bc-883a3c432236&currency=all&min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	api.GetExpensesReport(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "sql: expected 3 destination arguments in Scan, not 4", genericResp.Message)

	// (200)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT SUM.*").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(20000))
	dbMock.ExpectQuery("SELECT SUM.*").WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(35))
	dbMock.ExpectQuery("SELECT e.currency.*").
		WillReturnRows(sqlmock.NewRows(label).
			AddRow("IDR", mockID, "test", 5000).
			AddRow("IDR", mockID, "test", 15000).
			AddRow("USD", mockID, "test", 15).
			AddRow("USD", mockID, "test", 20))

	req, _ = http.NewRequest("GET", "?category_id=3e80f025-ff3c-4b25-a7bc-883a3c432236&currency=all&min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetExpensesReport(c)

	var report models.ExpenseReport

	err = json.NewDecoder(w.Body).Decode(&report)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, float64(20000), report.TotalIdr)
	assert.Equal(t, float64(35), report.TotalUsd)
	assert.Equal(t, 2, len(report.ReportsIdr))
	assert.Equal(t, 2, len(report.ReportsUsd))

}

func TestGetExpenses(t *testing.T) {
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

	dbMock.ExpectQuery("SELECT e.id.*").WillReturnError(errors.New("err-select"))

	req, _ := http.NewRequest("GET", "", nil)
	c.Request = req
	api.GetExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// scan error (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	label := []string{
		"id",
		"category_id",
		"name",
		"description",
		"id",
		"name",
		"description",
		"date",
		"currency",
		"amount",
		"user_id",
		"created_at",
		"updated_at",
	}

	dbMock.ExpectQuery("SELECT e.id.*").
		WillReturnRows(sqlmock.NewRows(label[10:]).AddRow(mockUserID, time.Now(), time.Now()))

	req, _ = http.NewRequest("GET", "?order_by=id", nil)
	c.Request = req
	api.GetExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "sql: expected 3 destination arguments in Scan, not 13", genericResp.Message)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT e.id.*").WithArgs(mockUserID).
		WillReturnRows(sqlmock.NewRows(label).
			AddRow(mockID, mockID, "dummy", "dummy", mockID,
				"dummy", "dummy", time.Now(), "IDR", 25555,
				mockUserID, time.Now(), time.Now()))
	dbMock.ExpectQuery("SELECT COUNT.*").WillReturnError(errors.New("err-count"))

	req, _ = http.NewRequest("GET", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-count", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT e.id.*").
		WillReturnRows(sqlmock.NewRows(label).
			AddRow(mockID, mockID, "dummy", "dummy", mockID,
				"dummy", "dummy", time.Now(), "IDR", 25555,
				mockUserID, time.Now(), time.Now()))
	dbMock.ExpectQuery("SELECT COUNT.*").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	q := url.Values{}
	q.Add("category_id", mockID)
	q.Add("product_id", mockID)
	q.Add("product_name", "dummy")
	q.Add("currency", "IDR")
	q.Add("amount", "2555")
	q.Add("date", "2000-01-01")
	q.Add("min_date", "2000-01-01")
	q.Add("max_date", "2050-01-01")
	q.Add("min_amount", "2555")
	q.Add("max_amount", "5555")

	req, _ = http.NewRequest("GET", "", nil)
	req.URL.RawQuery = q.Encode()
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetExpenses(c)

	var resp models.ExpenseList
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, int(resp.Total))
	assert.Equal(t, 1, len(resp.Expenses))
	assert.Equal(t, mockID, resp.Expenses[0].Id)
	assert.Equal(t, mockUserID, resp.Expenses[0].UserId)

	// as excel
	// expenses not found (404)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT e.id.*").WithArgs(mockUserID).
		WillReturnRows(sqlmock.NewRows(label))

	req, _ = http.NewRequest("GET", "?export_as_excel=true", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "expenses-not-found", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT e.id.*").
		WillReturnRows(sqlmock.NewRows(label).
			AddRow(mockID, mockID, "dummy", "dummy", mockID,
				"dummy", "dummy", time.Now(), "IDR", 25555,
				mockUserID, time.Now(), time.Now()))
	req, _ = http.NewRequest("GET", "?export_as_excel=true", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetExpenses(c)

	loc, _ := time.LoadLocation("Asia/Jakarta")
	fileName := fmt.Sprintf("report_expenses_%s.xlsx", time.Now().In(loc).Format("20060102_150405"))

	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "attachment;filename=\""+fileName+"\"", w.Header()["Content-Disposition"][0])

}

func TestUpsertExpenses(t *testing.T) {
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
	api.UpsertExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid request", genericResp.Message)

	// bad request (400)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload := parsePayload(models.UpsertExpenseRequest{})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpsertExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-expenses", genericResp.Message)

	// err begin (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.UpsertExpenseRequest{Data: []models.Expense{
		{},
	}})

	dbMock.ExpectBegin().WillReturnError(fmt.Errorf("err-begin"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpsertExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-begin", genericResp.Message)

	// expenses validation & insert failure (500)
	respErrors := struct {
		Message string            `json:"message"`
		Details []models.RowError `json:"details"`
	}{}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	mockID := "63eb226a-d612-412b-b8d4-a3e17b7d2226"
	mockID2 := "63eb226a-d612-412b-b8d4-a3e17b7d2227"
	mockUserID := "63eb226a-d612-412b-b8d4-a3e17b7d2228"
	dateFuture := time.Now().AddDate(1, 1, 1)
	expenses := models.UpsertExpenseRequest{Data: []models.Expense{
		{},
		{Id: mockID, ProductId: "product 1"},
		{Id: mockID, ProductId: "product 1", Date: "Y"},
		{Id: mockID, ProductId: "product 1", Date: "Y", Currency: "asd"},
		{Id: mockID, ProductId: "product 1", Date: "Y", Currency: "asd", Amount: 5555, UserId: "err"},
		{Id: mockID, ProductId: "product 1", Date: "Y", Currency: "asd", Amount: 5555, UserId: mockUserID},
		{Id: mockID, ProductId: mockID, Date: "Y", Currency: "asd", Amount: 5555, UserId: mockUserID},
		{Id: mockID, ProductId: mockID, Date: dateFuture.Format("2006-01-02"), Currency: "asd", Amount: 5555, UserId: mockUserID},
		{Id: mockID, ProductId: mockID, Date: "2000-01-01", Currency: "asd", Amount: 5555, UserId: mockUserID},
		{Id: mockID, ProductId: mockID, Date: "2000-01-01", Currency: "USD", Amount: 5555, UserId: mockUserID},
	}}
	dataOK := expenses.Data[len(expenses.Data)-1]
	payload = parsePayload(expenses)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO expenses.*").
		WithArgs(dataOK.Id, dataOK.ProductId, dataOK.Date, dataOK.UserId, dataOK.Currency, dataOK.Amount).WillReturnError(fmt.Errorf("err-insert"))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"ADMIN\"}}")
	api.UpsertExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&respErrors)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "error", respErrors.Message)
	assert.Equal(t, 10, len(respErrors.Details))
	assert.Equal(t, "missing-product-id", respErrors.Details[0].Message)
	assert.Equal(t, "missing-date", respErrors.Details[1].Message)
	assert.Equal(t, "missing-currency", respErrors.Details[2].Message)
	assert.Equal(t, "missing-amount", respErrors.Details[3].Message)
	assert.Equal(t, "invalid-user-id", respErrors.Details[4].Message)
	assert.Equal(t, "invalid-product-id", respErrors.Details[5].Message)
	assert.Equal(t, "invalid-date(yyyy-mm-dd)", respErrors.Details[6].Message)
	assert.Equal(t, "date-shall-be-a-past-date", respErrors.Details[7].Message)
	assert.Equal(t, "only-usd-or-idr-currency-are-allowed", respErrors.Details[8].Message)
	assert.Equal(t, "err-insert", respErrors.Details[9].Message)

	// err commit (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	expenses = models.UpsertExpenseRequest{Data: []models.Expense{
		{Id: mockID, ProductId: mockID, Date: "2000-01-01", Currency: "USD", Amount: 5555, UserId: mockUserID},
		{Id: mockID2, ProductId: mockID, Date: "2000-01-01", Currency: "IDR", Amount: 5555, UserId: mockUserID},
		{Id: mockID, ProductId: mockID, Date: "2000-01-01", Currency: "IDR", Amount: 5555, UserId: mockUserID},
	}}
	payload = parsePayload(expenses)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO expenses.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO expenses.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO expenses.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit().WillReturnError(fmt.Errorf("err-commit"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.UpsertExpenses(c)

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
	dbMock.ExpectExec("INSERT INTO expenses.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO expenses.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO expenses.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit()

	payload = parsePayload(expenses)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.UpsertExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&respSuccess)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", respSuccess.Message)
	assert.Equal(t, 3, respSuccess.Total)
}

func TestDeleteExpenses(t *testing.T) {
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
	api.DeleteExpenses(c)

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
	api.DeleteExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-data", genericResp.Message)

	mockID := "63eb226a-d612-412b-b8d4-a3e17b7d2226"
	// bad request (400)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{"error"}})

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteExpenses(c)

	var rowResp models.RowResponseError

	err = json.NewDecoder(w.Body).Decode(&rowResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, 1, len(rowResp.Detail))
	assert.Equal(t, 0, rowResp.Detail[0].Row)
	assert.Equal(t, "invalid-id", rowResp.Detail[0].Message)

	// err begin (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{mockID, mockID}})

	dbMock.ExpectBegin().WillReturnError(fmt.Errorf("err-begin"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-begin", genericResp.Message)

	// exec error (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{mockID, mockID}})

	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE expenses.*").WillReturnError(fmt.Errorf("err-exec"))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteExpenses(c)

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
	dbMock.ExpectExec("UPDATE expenses.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteExpenses(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, fmt.Sprintf("expected-%d-deleted-but-got-%d", len(reqData.Data), 1), genericResp.Message)

	// err commit (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqData)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE expenses.*").WillReturnResult(sqlmock.NewResult(0, 2))
	dbMock.ExpectCommit().WillReturnError(fmt.Errorf("err-commit"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteExpenses(c)

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
	dbMock.ExpectExec("UPDATE expenses.*").WillReturnResult(sqlmock.NewResult(0, 2))
	dbMock.ExpectCommit()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.DeleteExpenses(c)

	var respOk map[string]string

	err = json.NewDecoder(w.Body).Decode(&respOk)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", respOk["message"])
}
