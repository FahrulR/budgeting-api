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

func TestGetIncomesReporters(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	assert.Equal(t, nil, err)

	api := NewAPI()
	api.Db = db

	var genericResp GenericResponse

	// err select (500)
	mockUserID := "63eb226a-d612-412b-b8d4-a3e17b7d2227"

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT currency.*").WillReturnError(errors.New("err-select"))

	req, _ := http.NewRequest("GET", "?min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	api.GetIncomesReport(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// err scan (500)
	label := []string{"currency", "total"}
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT currency.*").WillReturnRows(sqlmock.NewRows(label[1:]).AddRow(1234))

	req, _ = http.NewRequest("GET", "?min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	api.GetIncomesReport(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "sql: expected 1 destination arguments in Scan, not 2", genericResp.Message)

	// (200)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT currency.*").
		WillReturnRows(sqlmock.NewRows(label).
			AddRow("IDR", 5000).
			AddRow("USD", 20))

	req, _ = http.NewRequest("GET", "?min_date=2020-01-01&max_date=2020-02-02", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetIncomesReport(c)

	var report models.IncomeReport

	err = json.NewDecoder(w.Body).Decode(&report)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, float64(5000), report.TotalIdr)
	assert.Equal(t, float64(20), report.TotalUsd)
}

func TestGetIncomes(t *testing.T) {
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

	dbMock.ExpectQuery("SELECT id.*").WillReturnError(errors.New("err-select"))

	req, _ := http.NewRequest("GET", "", nil)
	c.Request = req
	api.GetIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-select", genericResp.Message)

	// scan error (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	label := []string{
		"id",
		"name",
		"description",
		"user_id",
		"date",
		"currency",
		"amount",
		"created_at",
		"updated_at",
	}

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows(label[6:]).AddRow(100, time.Now(), time.Now()))

	req, _ = http.NewRequest("GET", "?order_by=id", nil)
	c.Request = req
	api.GetIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "sql: expected 3 destination arguments in Scan, not 9", genericResp.Message)

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT id.*").WithArgs(mockUserID).
		WillReturnRows(sqlmock.NewRows(label).
			AddRow(mockID, "name", "desc",
				mockUserID, time.Now(), "USD",
				5001, time.Now(), time.Now()))
	dbMock.ExpectQuery("SELECT COUNT.*").WillReturnError(errors.New("err-count"))

	req, _ = http.NewRequest("GET", "", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-count", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows(label).
			AddRow(mockID, "name", "desc",
				mockUserID, time.Now(), "USD",
				5001, time.Now(), time.Now()))
	dbMock.ExpectQuery("SELECT COUNT.*").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	q := url.Values{}
	q.Add("name", "dummy")
	q.Add("description", "dummy")
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
	api.GetIncomes(c)

	var resp models.IncomeList
	err = json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, int(resp.Total))
	assert.Equal(t, 1, len(resp.Incomes))
	assert.Equal(t, mockID, resp.Incomes[0].Id)
	assert.Equal(t, mockUserID, resp.Incomes[0].UserId)

	// as excel
	// incomes not found (404)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT id.*").WithArgs(mockUserID).
		WillReturnRows(sqlmock.NewRows(label))

	req, _ = http.NewRequest("GET", "?export_as_excel=true", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "incomes-not-found", genericResp.Message)

	// 200
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)

	dbMock.ExpectQuery("SELECT id.*").
		WillReturnRows(sqlmock.NewRows(label).
			AddRow(mockID, "name", "desc",
				mockUserID, time.Now(), "IDR",
				5001, time.Now(), time.Now()))
	req, _ = http.NewRequest("GET", "?export_as_excel=true", nil)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.GetIncomes(c)

	loc, _ := time.LoadLocation("Asia/Jakarta")
	fileName := fmt.Sprintf("report_incomes_%s.xlsx", time.Now().In(loc).Format("20060102_150405"))

	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "attachment;filename=\""+fileName+"\"", w.Header()["Content-Disposition"][0])

}

func TestUpsertIncomes(t *testing.T) {
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
	api.UpsertIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "invalid request", genericResp.Message)

	// bad request (400)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload := parsePayload(models.UpsertIncomeRequest{})
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpsertIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "missing-incomes", genericResp.Message)

	// err begin (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.UpsertIncomeRequest{Data: []models.Income{
		{},
	}})

	dbMock.ExpectBegin().WillReturnError(fmt.Errorf("err-begin"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.UpsertIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-begin", genericResp.Message)

	// incomes validation & insert failure (500)
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
	incomes := models.UpsertIncomeRequest{Data: []models.Income{
		{},
		{Id: mockID, Name: "income 1"},
		{Id: mockID, Name: "income 1", Description: "income desc 1"},
		{Id: mockID, Name: "income 1", Description: "income desc 1", Date: "err"},
		{Id: mockID, Name: "income 1", Description: "income desc 1", Date: "err", Currency: "err"},
		{Id: mockID, Name: "income 1", Description: "income desc 1", Date: "err", Currency: "err", Amount: 555, UserId: "err"},
		{Id: mockID, Name: "income 1", Description: "income desc 1", Date: "err", Currency: "err", Amount: 555, UserId: mockUserID},
		{Id: mockID, Name: "income 1", Description: "income desc 1", Date: dateFuture.Format("2006-01-02"), Currency: "err", Amount: 555, UserId: mockUserID},
		{Id: mockID, Name: "income 1", Description: "income desc 1", Date: "2000-01-02", Currency: "err", Amount: 555, UserId: mockUserID},
		{Id: mockID, Name: "income 1", Description: "income desc 1", Date: "2000-01-02", Currency: "USD", Amount: 555, UserId: mockUserID},
	}}
	dataOK := incomes.Data[len(incomes.Data)-1]
	payload = parsePayload(incomes)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO incomes.*").
		WithArgs(dataOK.Id, dataOK.Name, dataOK.Description, dataOK.Date, dataOK.UserId, dataOK.Currency, dataOK.Amount).WillReturnError(fmt.Errorf("err-insert"))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"ADMIN\"}}")
	api.UpsertIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&respErrors)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "error", respErrors.Message)
	assert.Equal(t, 10, len(respErrors.Details))
	assert.Equal(t, "missing-name", respErrors.Details[0].Message)
	assert.Equal(t, "missing-description", respErrors.Details[1].Message)
	assert.Equal(t, "missing-date", respErrors.Details[2].Message)
	assert.Equal(t, "missing-currency", respErrors.Details[3].Message)
	assert.Equal(t, "missing-amount", respErrors.Details[4].Message)
	assert.Equal(t, "invalid-user-id", respErrors.Details[5].Message)
	assert.Equal(t, "invalid-date(yyyy-mm-dd)", respErrors.Details[6].Message)
	assert.Equal(t, "date-shall-be-a-past-date", respErrors.Details[7].Message)
	assert.Equal(t, "only-usd-or-idr-currency-are-allowed", respErrors.Details[8].Message)
	assert.Equal(t, "err-insert", respErrors.Details[9].Message)

	// err commit (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	incomes = models.UpsertIncomeRequest{Data: []models.Income{
		{Id: mockID, Name: "income 1", Description: "income desc 1", Date: "2000-01-02", Currency: "USD", Amount: 555, UserId: mockUserID},
		{Id: mockID2, Name: "income 1", Description: "income desc 1", Date: "2000-01-02", Currency: "USD", Amount: 555, UserId: mockUserID},
		{Id: mockID, Name: "income 1", Description: "income desc 1", Date: "2000-01-02", Currency: "USD", Amount: 555, UserId: mockUserID},
	}}
	payload = parsePayload(incomes)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("INSERT INTO incomes.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO incomes.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO incomes.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit().WillReturnError(fmt.Errorf("err-commit"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.UpsertIncomes(c)

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
	dbMock.ExpectExec("INSERT INTO incomes.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO incomes.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectExec("INSERT INTO incomes.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectCommit()

	payload = parsePayload(incomes)
	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.UpsertIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&respSuccess)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "success", respSuccess.Message)
	assert.Equal(t, 3, respSuccess.Total)
}

func TestDeleteIncomes(t *testing.T) {
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
	api.DeleteIncomes(c)

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
	api.DeleteIncomes(c)

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
	api.DeleteIncomes(c)

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
	api.DeleteIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "err-begin", genericResp.Message)

	// exec error (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(models.BatchDeleteRequest{Data: []string{mockID, mockID}})

	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE incomes.*").WillReturnError(fmt.Errorf("err-exec"))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteIncomes(c)

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
	dbMock.ExpectExec("UPDATE incomes.*").WillReturnResult(sqlmock.NewResult(0, 1))
	dbMock.ExpectRollback()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteIncomes(c)

	err = json.NewDecoder(w.Body).Decode(&genericResp)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, fmt.Sprintf("expected-%d-deleted-but-got-%d", len(reqData.Data), 1), genericResp.Message)

	// err commit (500)
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	payload = parsePayload(reqData)

	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE incomes.*").WillReturnResult(sqlmock.NewResult(0, 2))
	dbMock.ExpectCommit().WillReturnError(fmt.Errorf("err-commit"))

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	api.DeleteIncomes(c)

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
	dbMock.ExpectExec("UPDATE incomes.*").WillReturnResult(sqlmock.NewResult(0, 2))
	dbMock.ExpectCommit()

	req, _ = http.NewRequest("POST", "", payload)
	c.Request = req
	c.Request.Header.Set("payload", "{\"user\":{\"id\":\""+mockUserID+"\", \"role\":\"CUSTOMER\"}}")
	api.DeleteIncomes(c)

	var respOk map[string]string

	err = json.NewDecoder(w.Body).Decode(&respOk)
	assert.Equal(t, nil, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", respOk["message"])
}
