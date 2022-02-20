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

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

func (api *API) GetExpenses(c *gin.Context) {
	u := ParsePayload(c)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	order := c.Query("order")
	orderBy := c.Query("order_by")

	amount, _ := strconv.ParseFloat(c.Query("amount"), 64)
	minAmount, _ := strconv.ParseFloat(c.Query("min_amount"), 64)
	maxAmount, _ := strconv.ParseFloat(c.Query("max_amount"), 64)

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
			e.id, p.category_id, c.name, p.id,
			p.name, p.description, e.date, e.currency,
			e.amount, e.user_id, e.created_at, e.updated_at
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

		var categoryId, categoryName, productId, productName,
			productDescription, currency, userId sql.NullString

		var amount sql.NullFloat64

		var date sql.NullTime

		err = rows.Scan(&expense.Id, &categoryId, &categoryName, &productId,
			&productName, &productDescription, &date, &currency,
			&amount, &userId, &expense.CreatedAt, &expense.UpdatedAt)
		if err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		expense.CategoryId = categoryId.String
		expense.CategoryName = categoryName.String
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
			log.Println(expense.Currency)
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

	if filter.Currency != "" {
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

	if len(expense.Currency) != 3 {
		return errors.New("invalid-currency")
	}

	expense.Currency = strings.ToUpper(expense.Currency)

	return nil
}

func (api *API) DeleteExpenses(c *gin.Context) {
	api.BatchDeletes(c, "expenses")
}
