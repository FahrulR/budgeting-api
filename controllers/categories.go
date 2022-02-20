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

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

func (api *API) GetCategories(c *gin.Context) {
	u := ParsePayload(c)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	order := c.Query("order")
	orderBy := c.Query("order_by")

	userId := c.Query("user_id")
	name := c.Query("name")
	description := c.Query("description")

	if u.Role == string(models.Customer) {
		userId = u.Id
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
		"created_at":  "created_at",
		"updated_at":  "updated_at",
	}

	if val, ok := mapOrderBy[orderBy]; ok {
		orderBy = val
	} else {
		orderBy = "updated_at"
	}

	countQ := `SELECT COUNT(1) FROM categories
		WHERE NOT deleted`
	selectQ := `SELECT
			id, name, description,
			user_id, created_at, updated_at
		FROM categories
		WHERE NOT deleted`

	var categoryList models.CategoryList
	var categories []models.Category
	var err error

	filterQ, stms := getFilterCategory(userId, name, description)

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
		var category models.Category
		var name, description, userId sql.NullString
		err = rows.Scan(&category.Id, &name, &description, &userId, &category.CreatedAt, &category.UpdatedAt)
		if err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		category.Name = name.String
		category.Description = description.String
		category.UserId = userId.String

		categories = append(categories, category)
	}

	categoryList.Total, err = api.GetTotal(countQ, stms)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	categoryList.Categories = categories
	categoryList.Limit = limit
	categoryList.Page = page

	c.JSON(http.StatusOK, categoryList)
}

func (api *API) UpsertCategories(c *gin.Context) {
	u := ParsePayload(c)
	var payload models.UpsertCategoryRequest

	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	categories := payload.Data
	if len(categories) == 0 {
		sendError(c, http.StatusBadRequest, "missing-categories")
		return
	}

	var errCategories []models.RowError
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

	for i, category := range categories {
		if customer {
			category.UserId = u.Id
		}

		if _, err := uuid.FromString(category.Id); err != nil {
			category.Id = uuid.Must(uuid.NewV4()).String()
		}

		if err := validateCategory(category); err != nil {
			errCategories = append(errCategories, models.RowError{Row: i + 1, Message: err.Error()})
			continue
		}

		if _, err := tx.Exec(`
		INSERT INTO categories
		(id, name, description, user_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
		name = $2, description = $3, user_id = $4, updated_at = CURRENT_TIMESTAMP, deleted = false
		`, category.Id, category.Name, category.Description, category.UserId); err != nil {
			log.Println(err)
			errCategories = append(errCategories, models.RowError{Row: i + 1, Message: err.Error()})
			continue
		}
	}

	code := http.StatusInternalServerError
	obj := gin.H{"message": "error", "details": errCategories}

	if len(errCategories) == 0 {
		if err := tx.Commit(); err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		code = http.StatusOK
		obj = gin.H{"message": "success", "total": len(categories)}
	}

	c.JSON(code, obj)
}

func (api *API) DeleteCategories(c *gin.Context) {
	api.BatchDeletes(c, "categories")
}

func validateCategory(category models.Category) error {

	if category.Name == "" {
		return errors.New("missing-name")
	}

	if category.Description == "" {
		return errors.New("missing-description")
	}

	if category.UserId == "" {
		return errors.New("missing-user-id")
	}

	if _, err := uuid.FromString(category.UserId); err != nil {
		return errors.New("invalid-user-id")
	}

	return nil
}

func getFilterCategory(userId, name, description string) (filterQ string, stms []interface{}) {
	if _, err := uuid.FromString(userId); err == nil {
		filterQ = fmt.Sprintf(" AND user_id = $%d", len(stms)+1)
		stms = append(stms, userId)
	}

	if name != "" {
		filterQ += fmt.Sprintf(" AND name ILIKE $%d", len(stms)+1)
		stms = append(stms, "%"+name+"%")
	}

	if description != "" {
		filterQ += fmt.Sprintf(" AND description ILIKE $%d", len(stms)+1)
		stms = append(stms, "%"+description+"%")
	}

	return
}
