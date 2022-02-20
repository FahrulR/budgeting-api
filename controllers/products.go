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
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

func (api *API) GetProducts(c *gin.Context) {
	u := ParsePayload(c)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))
	order := c.Query("order")
	orderBy := c.Query("order_by")

	userId := c.Query("user_id")
	name := c.Query("name")
	description := c.Query("description")
	categoryId := c.Query("category_id")

	asExcel, _ := strconv.ParseBool(c.Query("export_as_excel"))

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
		"id":          "p.id",
		"name":        "p.name",
		"description": "p.description",
		"user_id":     "p.user_id",
		"created_at":  "p.created_at",
		"updated_at":  "p.updated_at",
		"category":    "c.name",
	}

	if val, ok := mapOrderBy[orderBy]; ok {
		orderBy = val
	} else {
		orderBy = "p.updated_at"
	}

	countQ := `SELECT COUNT(1) FROM products p
		JOIN categories c
		ON p.category_id = c.id
		WHERE NOT p.deleted`
	selectQ := `SELECT
			p.id, p.name, p.description, p.user_id, p.category_id,
			p.created_at, p.updated_at, c.name, c.description
		FROM products p
		JOIN categories c
		ON p.category_id = c.id
		WHERE NOT p.deleted`

	var productList models.ProductList
	var products []models.Product
	var err error

	filterQ, stms := getFilterProduct(userId, name, description, categoryId)

	selectQ = selectQ + filterQ
	countQ = countQ + filterQ

	offset := (page - 1) * limit
	pagination := fmt.Sprintf(" LIMIT %d OFFSET %d ", limit, offset)
	orderVal := fmt.Sprintf(" ORDER BY %s %s", orderBy, order)

	log.Println(selectQ + orderVal + pagination)

	products, err = api.getProducts(selectQ+orderVal+pagination, stms)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if asExcel {
		handleExcelProducts(c, products)
		return
	}

	productList.Total, err = api.GetTotal(countQ, stms)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	productList.Products = products
	productList.Limit = limit
	productList.Page = page

	c.JSON(http.StatusOK, productList)
}

func (api *API) UpsertProducts(c *gin.Context) {
	u := ParsePayload(c)
	var payload models.UpsertProductRequest

	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	products := payload.Data
	if len(products) == 0 {
		sendError(c, http.StatusBadRequest, "missing-products")
		return
	}

	var errProducts []models.RowError
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

	for i, product := range products {
		if customer {
			product.UserId = u.Id
		}

		if _, err := uuid.FromString(product.Id); err != nil {
			product.Id = uuid.Must(uuid.NewV4()).String()
		}

		if err := validateProduct(product); err != nil {
			errProducts = append(errProducts, models.RowError{Row: i + 1, Message: err.Error()})
			continue
		}

		if _, err := tx.Exec(`
		INSERT INTO products
		(id, name, description, user_id, created_at, updated_at, category_id)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, $5)
		ON CONFLICT(id) DO UPDATE SET
		name = $2, description = $3, user_id = $4, updated_at = CURRENT_TIMESTAMP, deleted = false, category_id = $5
		`, product.Id, product.Name, product.Description, product.UserId, product.CategoryId); err != nil {
			log.Println(err)
			errProducts = append(errProducts, models.RowError{Row: i + 1, Message: err.Error()})
			continue
		}
	}

	code := http.StatusInternalServerError
	obj := gin.H{"message": "error", "details": errProducts}

	if len(errProducts) == 0 {
		if err := tx.Commit(); err != nil {
			log.Println(err)
			sendError(c, http.StatusInternalServerError, err.Error())
			return
		}

		code = http.StatusOK
		obj = gin.H{"message": "success", "total": len(products)}
	}

	c.JSON(code, obj)
}

func (api *API) DeleteProducts(c *gin.Context) {
	api.BatchDeletes(c, "products")
}

func handleExcelProducts(c *gin.Context, products []models.Product) {
	if len(products) == 0 {
		sendError(c, http.StatusNotFound, "products-not-found")
		return
	}

	f := excelize.NewFile()

	sheet := "List Products"
	f.NewSheet(sheet)
	// delete default sheet
	f.DeleteSheet("Sheet1")

	err := f.SetColWidth(sheet, "A", "E", 50)
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
		excelize.Cell{StyleID: headerStyle, Value: "Name"},
		excelize.Cell{StyleID: headerStyle, Value: "Description"},
		excelize.Cell{StyleID: headerStyle, Value: "Created At"},
		excelize.Cell{StyleID: headerStyle, Value: "Updated At"}}); err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")

	for n, product := range products {
		createdAt := product.CreatedAt.In(loc).Format("2006-01-02 15:04:05")
		updatedAt := product.UpdatedAt.In(loc).Format("2006-01-02 15:04:05")

		if updatedAt == createdAt {
			updatedAt = "-"
		}

		row := make([]interface{}, 5)
		row[0] = excelize.Cell{StyleID: dataStyle, Value: product.CategoryName}
		row[1] = excelize.Cell{StyleID: dataStyle, Value: product.Name}
		row[2] = excelize.Cell{StyleID: dataStyle, Value: product.Description}
		row[3] = excelize.Cell{StyleID: dataStyle, Value: createdAt}
		row[4] = excelize.Cell{StyleID: dataStyle, Value: updatedAt}

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

	fileName := fmt.Sprintf("report_products_%s.xlsx", time.Now().In(loc).Format("20060102_150405"))

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment;filename=\""+fileName+"\"")

	if _, err := f.WriteTo(c.Writer); err != nil {
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

}

func (api *API) getProducts(q string, stms []interface{}) (products []models.Product, err error) {

	rows, err := api.Db.Query(q, stms...)
	if err != nil {
		log.Println(err)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var product models.Product
		var name, description, userId, categoryId, categoryName, categoryDescription sql.NullString
		err = rows.Scan(&product.Id, &name, &description, &userId, &categoryId, &product.CreatedAt, &product.UpdatedAt, &categoryName, &categoryDescription)
		if err != nil {
			log.Println(err)
			return
		}

		product.Name = name.String
		product.Description = description.String
		product.UserId = userId.String
		product.CategoryId = categoryId.String
		product.CategoryName = categoryName.String
		product.CategoryDescription = categoryDescription.String

		products = append(products, product)
	}

	return

}

func getFilterProduct(userId, name, description, categoryId string) (filterQ string, stms []interface{}) {
	if _, err := uuid.FromString(userId); err == nil {
		filterQ = fmt.Sprintf(" AND p.user_id = $%d", len(stms)+1)
		stms = append(stms, userId)
	}

	if name != "" {
		filterQ += fmt.Sprintf(" AND p.name ILIKE $%d", len(stms)+1)
		stms = append(stms, "%"+name+"%")
	}

	if description != "" {
		filterQ += fmt.Sprintf(" AND p.description ILIKE $%d", len(stms)+1)
		stms = append(stms, "%"+description+"%")
	}

	if _, err := uuid.FromString(categoryId); err == nil {
		filterQ += fmt.Sprintf(" AND p.category_id = $%d", len(stms)+1)
		stms = append(stms, categoryId)
	}

	return
}

func validateProduct(product models.Product) error {

	if product.Name == "" {
		return errors.New("missing-name")
	}

	if product.Description == "" {
		return errors.New("missing-description")
	}

	if product.UserId == "" {
		return errors.New("missing-user-id")
	}

	if _, err := uuid.FromString(product.UserId); err != nil {
		return errors.New("invalid-user-id")
	}

	if _, err := uuid.FromString(product.CategoryId); err != nil {
		return errors.New("invalid-category-id")
	}

	return nil
}
