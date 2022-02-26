package controllers

import (
	"budgetingapi/models"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

func (api *API) GetIncomesReport(c *gin.Context) {
	u := ParsePayload(c)
	filter := models.IncomeFilter{
		Income: models.Income{
			UserId: c.Query("user_id"),
		},
		MinDate: c.Query("min_date"),
		MaxDate: c.Query("max_date"),
	}

	if u.Role == string(models.Customer) {
		filter.UserId = u.Id
	}

	totalQ := `SELECT currency, SUM(amount) FROM incomes e WHERE NOT deleted`
	filterQ, stms := getFilterIncome(filter)

	groupBy := `GROUP BY currency`

	totalQ = totalQ + filterQ + groupBy

	var report models.IncomeReport

	rows, err := api.Db.Query(totalQ, stms...)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var currency sql.NullString
		var total sql.NullFloat64

		if err := rows.Scan(&currency, &total); err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		if currency.String == "IDR" {
			report.TotalIdr = total.Float64
		}

		if currency.String == "USD" {
			report.TotalUsd = total.Float64
		}
	}

	c.JSON(http.StatusOK, report)
}

func (api *API) GetIncomes(c *gin.Context) {
	u := ParsePayload(c)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	order := c.Query("order")
	orderBy := c.Query("order_by")

	amount, _ := strconv.ParseFloat(c.Query("amount"), 64)
	minAmount, _ := strconv.ParseFloat(c.Query("min_amount"), 64)
	maxAmount, _ := strconv.ParseFloat(c.Query("max_amount"), 64)

	asExcel, _ := strconv.ParseBool(c.Query("export_as_excel"))

	filter := models.IncomeFilter{
		Income: models.Income{
			Name:        c.Query("name"),
			Description: c.Query("description"),
			UserId:      c.Query("user_id"),
			Currency:    c.Query("currency"),
			Amount:      amount,
			Date:        c.Query("date"),
		},
		MinDate:   c.Query("min_date"),
		MaxDate:   c.Query("max_date"),
		MinAmount: minAmount,
		MaxAmount: maxAmount,
	}

	if u.Role == string(models.Customer) {
		filter.UserId = u.Id
	}

	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 20
	}

	if strings.ToUpper(order) != "ASC" && strings.ToUpper(order) != "DESC" {
		order = "DESC"
	}

	mapOrderBy := map[string]string{
		"id":          "id",
		"name":        "name",
		"description": "description",
		"user_id":     "user_id",
		"date":        "date",
		"currency":    "currency",
		"amount":      "amount",
		"created_at":  "created_at",
		"updated_at":  "updated_at",
	}

	if val, ok := mapOrderBy[orderBy]; ok {
		orderBy = val
	} else {
		orderBy = "updated_at"
	}

	countQ := `SELECT COUNT(1) FROM incomes
		WHERE NOT deleted`
	selectQ := `SELECT
			id, name, description,
			user_id, date, currency,
			amount, created_at, updated_at
		FROM incomes
		WHERE NOT deleted`

	var incomeList models.IncomeList
	var incomes []models.Income
	var err error

	filterQ, stms := getFilterIncome(filter)

	selectQ = selectQ + filterQ
	countQ = countQ + filterQ

	offset := (page - 1) * limit
	pagination := fmt.Sprintf(" LIMIT %d OFFSET %d ", limit, offset)
	orderVal := fmt.Sprintf(" ORDER BY %s %s", orderBy, order)

	log.Println(selectQ + orderVal + pagination)

	rows, err := api.Db.Query(selectQ+orderVal+pagination, stms...)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var income models.Income
		var name, description, userId, currency sql.NullString

		var amount sql.NullFloat64

		var date sql.NullTime

		err = rows.Scan(&income.Id, &name, &description, &userId, &date, &currency, &amount, &income.CreatedAt, &income.UpdatedAt)
		if err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		income.Name = name.String
		income.Description = description.String
		income.UserId = userId.String

		if date.Valid {
			income.Date = date.Time.Format(dateFormat)
		}

		income.Currency = currency.String
		income.Amount = amount.Float64

		incomes = append(incomes, income)
	}

	if asExcel {
		handleExcelIncomes(c, incomes)
		return
	}

	incomeList.Total, err = api.GetTotal(countQ, stms)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	incomeList.Incomes = incomes
	incomeList.Limit = limit
	incomeList.Page = page

	c.JSON(http.StatusOK, incomeList)
}

func (api *API) UpsertIncomes(c *gin.Context) {
	u := ParsePayload(c)
	var payload models.UpsertIncomeRequest

	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	incomes := payload.Data
	if len(incomes) == 0 {
		sendError(c, http.StatusBadRequest, "missing-incomes")
		return
	}

	var errIncomes []models.RowError
	tx, err := api.Db.Begin()
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	defer tx.Rollback()

	customer := false
	if u.Role == string(models.Customer) {
		customer = true
	}

	for i, income := range incomes {
		if customer {
			income.UserId = u.Id
		}

		if _, err := uuid.FromString(income.Id); err != nil {
			income.Id = uuid.Must(uuid.NewV4()).String()
		}

		if err := validateIncome(&income); err != nil {
			errIncomes = append(errIncomes, models.RowError{Row: i + 1, Message: err.Error()})
			continue
		}

		if _, err := tx.Exec(`
		INSERT INTO incomes
		(id, name, description, date, user_id, currency, amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		name = $2, description = $3, date = $4, user_id = $5, currency = $6, amount= $7, updated_at = CURRENT_TIMESTAMP, deleted = false
		`, income.Id, income.Name, income.Description, income.Date, income.UserId, income.Currency, income.Amount); err != nil {
			log.Println(err)
			errIncomes = append(errIncomes, models.RowError{Row: i + 1, Message: err.Error()})
			continue
		}
	}

	code := http.StatusInternalServerError
	obj := gin.H{"message": "error", "details": errIncomes}

	if len(errIncomes) == 0 {
		if err := tx.Commit(); err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		code = http.StatusOK
		obj = gin.H{"message": "success", "total": len(incomes)}
	}

	c.JSON(code, obj)
}

func (api *API) DeleteIncomes(c *gin.Context) {
	api.BatchDeletes(c, "incomes")
}

func handleExcelIncomes(c *gin.Context, incomes []models.Income) {
	if len(incomes) == 0 {
		sendError(c, http.StatusNotFound, "incomes-not-found")
		return
	}

	f := excelize.NewFile()

	sheet := "List Incomes"
	f.NewSheet(sheet)
	// delete default sheet
	f.DeleteSheet("Sheet1")

	err := f.SetColWidth(sheet, "A", "F", 50)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	headerStyle, err := f.NewStyle(s1)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	dataStyle, err := f.NewStyle(s2)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	streamWriter, err := f.NewStreamWriter(sheet)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err = streamWriter.SetRow("A1", []interface{}{
		excelize.Cell{StyleID: headerStyle, Value: "Name"},
		excelize.Cell{StyleID: headerStyle, Value: "Description"},
		excelize.Cell{StyleID: headerStyle, Value: "Currency"},
		excelize.Cell{StyleID: headerStyle, Value: "Amount"},
		excelize.Cell{StyleID: headerStyle, Value: "Date"}}); err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	for n, income := range incomes {
		amountFormatted := fmt.Sprintf("$%s", humanize.Commaf(income.Amount))
		if income.Currency == "IDR" {
			amountFormatted = strings.ReplaceAll(fmt.Sprintf("Rp %s", humanize.Commaf(income.Amount)), ",", ".")
		}

		row := make([]interface{}, 5)
		row[0] = excelize.Cell{StyleID: dataStyle, Value: income.Name}
		row[1] = excelize.Cell{StyleID: dataStyle, Value: income.Description}
		row[2] = excelize.Cell{StyleID: dataStyle, Value: income.Currency}
		row[3] = excelize.Cell{StyleID: dataStyle, Value: amountFormatted}
		row[4] = excelize.Cell{StyleID: dataStyle, Value: income.Date}

		cell, _ := excelize.CoordinatesToCellName(1, n+2)
		if err = streamWriter.SetRow(cell, row); err != nil {
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := streamWriter.Flush(); err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	fileName := fmt.Sprintf("report_incomes_%s.xlsx", time.Now().In(loc).Format("20060102_150405"))

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment;filename=\""+fileName+"\"")

	if _, err := f.WriteTo(c.Writer); err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

}

func getFilterIncome(filter models.IncomeFilter) (filterQ string, stms []interface{}) {
	if _, err := uuid.FromString(filter.UserId); err == nil {
		filterQ = fmt.Sprintf(" AND user_id = $%d", len(stms)+1)
		stms = append(stms, filter.UserId)
	}

	if filter.Name != "" {
		filterQ += fmt.Sprintf(" AND name ILIKE $%d", len(stms)+1)
		stms = append(stms, "%"+filter.Name+"%")
	}

	if filter.Description != "" {
		filterQ += fmt.Sprintf(" AND description ILIKE $%d", len(stms)+1)
		stms = append(stms, "%"+filter.Description+"%")
	}

	if filter.Currency == "IDR" || filter.Currency == "USD" {
		filterQ += fmt.Sprintf(" AND currency = $%d", len(stms)+1)
		stms = append(stms, filter.Currency)
	}

	if filter.Amount != 0 {
		filterQ += fmt.Sprintf(" AND amount = $%d", len(stms)+1)
		stms = append(stms, filter.Amount)
	}

	if date, err := time.Parse(dateFormat, filter.Date); err == nil {
		filterQ += fmt.Sprintf(" AND date = $%d", len(stms)+1)
		stms = append(stms, date)
	}

	if date, err := time.Parse(dateFormat, filter.MinDate); err == nil {
		filterQ += fmt.Sprintf(" AND date >= $%d", len(stms)+1)
		stms = append(stms, date)
	}

	if date, err := time.Parse(dateFormat, filter.MaxDate); err == nil {
		filterQ += fmt.Sprintf(" AND date <= $%d", len(stms)+1)
		stms = append(stms, date)
	}

	if filter.MinAmount != 0 {
		filterQ += fmt.Sprintf(" AND amount >= $%d", len(stms)+1)
		stms = append(stms, filter.MinAmount)
	}

	if filter.MaxAmount != 0 {
		filterQ += fmt.Sprintf(" AND amount <= $%d", len(stms)+1)
		stms = append(stms, filter.MaxAmount)
	}

	return
}

func validateIncome(income *models.Income) error {

	if income.Name == "" {
		return errors.New("missing-name")
	}

	if income.Description == "" {
		return errors.New("missing-description")
	}

	if income.Date == "" {
		return errors.New("missing-date")
	}

	if income.Currency == "" {
		return errors.New("missing-currency")
	}

	if income.Amount == 0 {
		return errors.New("missing-amount")
	}

	if _, err := uuid.FromString(income.UserId); err != nil {
		return errors.New("invalid-user-id")
	}

	date, err := time.Parse(dateFormat, income.Date)
	if err != nil {
		return errors.New("invalid-date(yyyy-mm-dd)")
	}

	if date.After(time.Now()) {
		return errors.New("date-shall-be-a-past-date")
	}

	// currently only allow USD and IDR
	income.Currency = strings.ToUpper(income.Currency)
	if income.Currency != "USD" && income.Currency != "IDR" {
		return errors.New("only-usd-or-idr-currency-are-allowed")
	}

	return nil
}
