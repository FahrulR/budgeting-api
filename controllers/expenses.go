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

func (api *API) GetExpensesReport(c *gin.Context) {
	u := ParsePayload(c)
	filter := models.ExpenseFilter{
		Expense: models.Expense{
			UserId:     c.Query("user_id"),
			CategoryId: c.Query("category_id"),
			Currency:   c.Query("currency"),
		},
		MinDate: c.Query("min_date"),
		MaxDate: c.Query("max_date"),
	}

	if u.Role == string(models.Customer) {
		filter.UserId = u.Id
	}

	totalQ := `SELECT SUM(amount) FROM expenses e
		JOIN products p ON e.product_id = p.id AND NOT p.deleted
		JOIN categories c ON p.category_id = c.id
		WHERE NOT e.deleted`

	selectQ := `SELECT e.currency, c.id, c.name, SUM(amount)
		FROM expenses e
		JOIN products p ON e.product_id = p.id AND NOT p.deleted
		JOIN categories c ON p.category_id = c.id
		WHERE NOT e.deleted`

	filterQ, stms := getFilterExpense(filter)

	selectQ = selectQ + filterQ
	totalQ = totalQ + filterQ

	var report models.ExpenseReport
	var err error

	report.TotalIdr, report.TotalUsd, err = api.getTotalIdrUsd(filter.Currency, totalQ, stms)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	qGroupBy := " GROUP BY e.currency, c.id, c.name"
	selectQ += qGroupBy

	log.Println(selectQ)

	rows, err := api.Db.Query(selectQ, stms...)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	defer rows.Close()

	for rows.Next() {
		var total sql.NullFloat64
		var categoryReport models.CategoryTotalReport
		var currency string

		if err := rows.Scan(&currency, &categoryReport.Id, &categoryReport.Name, &total); err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		categoryReport.Total = total.Float64

		if currency == "IDR" {
			report.ReportsIdr = append(report.ReportsIdr, categoryReport)
		}

		if currency == "USD" {
			report.ReportsUsd = append(report.ReportsUsd, categoryReport)
		}
	}

	c.JSON(http.StatusOK, report)
}

func (api *API) getTotalIdrUsd(currency, q string, stms []interface{}) (totalIdr, totalUsd float64, err error) {
	var sqlIdr, sqlUsd sql.NullFloat64

	if currency == "IDR" {
		err = api.Db.QueryRow(q, stms...).Scan(&sqlIdr)
		if err != nil {
			log.Println(err)
		}
		totalIdr = sqlIdr.Float64
		return
	}

	if currency == "USD" {
		err = api.Db.QueryRow(q, stms...).Scan(&sqlUsd)
		if err != nil {
			log.Println(err)
		}
		totalUsd = sqlUsd.Float64
		return
	}

	// get both
	q += fmt.Sprintf(" AND e.currency = $%d", len(stms)+1)
	stms = append(stms, "IDR")

	err = api.Db.QueryRow(q, stms...).Scan(&sqlIdr)
	if err != nil {
		log.Println(err)
		return
	}

	stms[len(stms)-1] = "USD"
	err = api.Db.QueryRow(q, stms...).Scan(&sqlUsd)
	if err != nil {
		log.Println(err)
		return
	}

	totalIdr = sqlIdr.Float64
	totalUsd = sqlUsd.Float64

	return
}

func (api *API) GetExpenses(c *gin.Context) {
	u := ParsePayload(c)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	order := c.Query("order")
	orderBy := c.Query("order_by")

	amount, _ := strconv.ParseFloat(c.Query("amount"), 64)
	minAmount, _ := strconv.ParseFloat(c.Query("min_amount"), 64)
	maxAmount, _ := strconv.ParseFloat(c.Query("max_amount"), 64)

	asExcel, _ := strconv.ParseBool(c.Query("export_as_excel"))

	filter := models.ExpenseFilter{
		Expense: models.Expense{
			UserId:      c.Query("user_id"),
			CategoryId:  c.Query("category_id"),
			ProductId:   c.Query("product_id"),
			ProductName: c.Query("product_name"),
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
		"id":            "e.id",
		"category_id":   "p.category_id",
		"category_name": "c.name",
		"product_name":  "p.name",
		"date":          "e.date",
		"currency":      "e.currency",
		"amount":        "e.amount",
		"user_id":       "e.user_id",
		"created_at":    "e.created_at",
		"updated_at":    "e.updated_at",
	}

	if val, ok := mapOrderBy[orderBy]; ok {
		orderBy = val
	} else {
		orderBy = "e.updated_at"
	}

	countQ := `SELECT COUNT(1) FROM expenses e
		JOIN products p ON e.product_id = p.id AND NOT p.deleted
		JOIN categories c ON p.category_id = c.id
		WHERE NOT e.deleted`
	selectQ := `SELECT
			e.id, p.category_id, c.name, c.description, p.id,
			p.name, p.description, e.date, e.currency, e.amount,
			e.user_id, e.created_at, e.updated_at
		FROM expenses e
		JOIN products p ON e.product_id = p.id AND NOT p.deleted
		JOIN categories c ON p.category_id = c.id
		WHERE NOT e.deleted`

	var expenseList models.ExpenseList
	var expenses []models.Expense
	var err error

	filterQ, stms := getFilterExpense(filter)

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
		var expense models.Expense

		var categoryId, categoryName, categoryDescription, productId,
			productName, productDescription, currency, userId sql.NullString

		var amount sql.NullFloat64

		var date sql.NullTime

		err = rows.Scan(&expense.Id, &categoryId, &categoryName, &categoryDescription, &productId,
			&productName, &productDescription, &date, &currency, &amount,
			&userId, &expense.CreatedAt, &expense.UpdatedAt)
		if err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		expense.CategoryId = categoryId.String
		expense.CategoryName = categoryName.String
		expense.CategoryDescription = categoryDescription.String
		expense.ProductId = productId.String
		expense.ProductName = productName.String
		expense.ProductDescription = productDescription.String
		expense.Currency = currency.String
		expense.Amount = amount.Float64
		expense.UserId = userId.String

		if date.Valid {
			expense.Date = date.Time.Format(dateFormat)
		}

		expenses = append(expenses, expense)
	}

	if asExcel {
		handleExcelExpenses(c, expenses)
		return
	}

	expenseList.Total, err = api.GetTotal(countQ, stms)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	expenseList.Expenses = expenses
	expenseList.Limit = limit
	expenseList.Page = page

	c.JSON(http.StatusOK, expenseList)
}

func (api *API) UpsertExpenses(c *gin.Context) {
	u := ParsePayload(c)
	var payload models.UpsertExpenseRequest

	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	expenses := payload.Data
	if len(expenses) == 0 {
		sendError(c, http.StatusBadRequest, "missing-expenses")
		return
	}

	var errExpenses []models.RowError
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

	for i, expense := range expenses {
		if customer {
			expense.UserId = u.Id
		}

		if _, err := uuid.FromString(expense.Id); err != nil {
			expense.Id = uuid.Must(uuid.NewV4()).String()
		}

		if err := validateExpense(&expense); err != nil {
			errExpenses = append(errExpenses, models.RowError{Row: i + 1, Message: err.Error()})
			continue
		}

		if _, err := tx.Exec(`
		INSERT INTO expenses
		(id, product_id, date, user_id, created_at, updated_at, currency, amount)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, $5, $6)
		ON CONFLICT(id) DO UPDATE SET
		product_id = $2, date = $3, user_id = $4, updated_at = CURRENT_TIMESTAMP, deleted = false, currency = $5, amount = $6
		`, expense.Id, expense.ProductId, expense.Date, expense.UserId, expense.Currency, expense.Amount); err != nil {
			log.Println(err)
			errExpenses = append(errExpenses, models.RowError{Row: i + 1, Message: err.Error()})
			continue
		}
	}

	code := http.StatusInternalServerError
	obj := gin.H{"message": "error", "details": errExpenses}

	if len(errExpenses) == 0 {
		if err := tx.Commit(); err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		code = http.StatusOK
		obj = gin.H{"message": "success", "total": len(expenses)}
	}

	c.JSON(code, obj)
}

func getFilterExpense(filter models.ExpenseFilter) (filterQ string, stms []interface{}) {
	if _, err := uuid.FromString(filter.UserId); err == nil {
		filterQ = fmt.Sprintf(" AND e.user_id = $%d", len(stms)+1)
		stms = append(stms, filter.UserId)
	}

	if _, err := uuid.FromString(filter.CategoryId); err == nil {
		filterQ += fmt.Sprintf(" AND p.category_id = $%d", len(stms)+1)
		stms = append(stms, filter.CategoryId)
	}

	if _, err := uuid.FromString(filter.ProductId); err == nil {
		filterQ += fmt.Sprintf(" AND e.product_id = $%d", len(stms)+1)
		stms = append(stms, filter.ProductId)
	}

	if filter.ProductName != "" {
		filterQ += fmt.Sprintf(" AND p.name ILIKE $%d", len(stms)+1)
		stms = append(stms, "%"+filter.ProductName+"%")
	}

	if filter.Currency == "IDR" || filter.Currency == "USD" {
		filterQ += fmt.Sprintf(" AND e.currency = $%d", len(stms)+1)
		stms = append(stms, filter.Currency)
	}

	if filter.Amount != 0 {
		filterQ += fmt.Sprintf(" AND e.amount = $%d", len(stms)+1)
		stms = append(stms, filter.Amount)
	}

	if date, err := time.Parse(dateFormat, filter.Date); err == nil {
		filterQ += fmt.Sprintf(" AND e.date = $%d", len(stms)+1)
		stms = append(stms, date)
	}

	if date, err := time.Parse(dateFormat, filter.MinDate); err == nil {
		filterQ += fmt.Sprintf(" AND e.date >= $%d", len(stms)+1)
		stms = append(stms, date)
	}

	if date, err := time.Parse(dateFormat, filter.MaxDate); err == nil {
		filterQ += fmt.Sprintf(" AND e.date <= $%d", len(stms)+1)
		stms = append(stms, date)
	}

	if filter.MinAmount != 0 {
		filterQ += fmt.Sprintf(" AND e.amount >= $%d", len(stms)+1)
		stms = append(stms, filter.MinAmount)
	}

	if filter.MaxAmount != 0 {
		filterQ += fmt.Sprintf(" AND e.amount <= $%d", len(stms)+1)
		stms = append(stms, filter.MaxAmount)
	}

	return
}

func validateExpense(expense *models.Expense) error {

	if expense.ProductId == "" {
		return errors.New("missing-product-id")
	}

	if expense.Date == "" {
		return errors.New("missing-date")
	}

	if expense.Currency == "" {
		return errors.New("missing-currency")
	}

	if expense.Amount == 0 {
		return errors.New("missing-amount")
	}

	if _, err := uuid.FromString(expense.UserId); err != nil {
		return errors.New("invalid-user-id")
	}

	if _, err := uuid.FromString(expense.ProductId); err != nil {
		return errors.New("invalid-product-id")
	}

	date, err := time.Parse(dateFormat, expense.Date)
	if err != nil {
		return errors.New("invalid-date(yyyy-mm-dd)")
	}

	if date.After(time.Now()) {
		return errors.New("date-shall-be-a-past-date")
	}

	// currently only allow USD and IDR
	expense.Currency = strings.ToUpper(expense.Currency)
	if expense.Currency != "USD" && expense.Currency != "IDR" {
		return errors.New("only-usd-or-idr-currency-are-allowed")
	}

	return nil
}

func (api *API) DeleteExpenses(c *gin.Context) {
	api.BatchDeletes(c, "expenses")
}

func handleExcelExpenses(c *gin.Context, expenses []models.Expense) {
	if len(expenses) == 0 {
		sendError(c, http.StatusNotFound, "expenses-not-found")
		return
	}

	f := excelize.NewFile()

	sheet := "List Expenses"
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
		excelize.Cell{StyleID: headerStyle, Value: "Category"},
		excelize.Cell{StyleID: headerStyle, Value: "Product Name"},
		excelize.Cell{StyleID: headerStyle, Value: "Product Description"},
		excelize.Cell{StyleID: headerStyle, Value: "Currency"},
		excelize.Cell{StyleID: headerStyle, Value: "Amount"},
		excelize.Cell{StyleID: headerStyle, Value: "Date"}}); err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	for n, expense := range expenses {
		amountFormatted := fmt.Sprintf("$%s", humanize.Commaf(expense.Amount))
		if expense.Currency == "IDR" {
			amountFormatted = strings.ReplaceAll(fmt.Sprintf("Rp %s", humanize.Commaf(expense.Amount)), ",", ".")
		}

		row := make([]interface{}, 6)
		row[0] = excelize.Cell{StyleID: dataStyle, Value: expense.CategoryName}
		row[1] = excelize.Cell{StyleID: dataStyle, Value: expense.ProductName}
		row[2] = excelize.Cell{StyleID: dataStyle, Value: expense.ProductDescription}
		row[3] = excelize.Cell{StyleID: dataStyle, Value: expense.Currency}
		row[4] = excelize.Cell{StyleID: dataStyle, Value: amountFormatted}
		row[5] = excelize.Cell{StyleID: dataStyle, Value: expense.Date}

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
	fileName := fmt.Sprintf("report_expenses_%s.xlsx", time.Now().In(loc).Format("20060102_150405"))

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment;filename=\""+fileName+"\"")

	if _, err := f.WriteTo(c.Writer); err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

}
