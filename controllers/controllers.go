package controllers

import (
	"budgetingapi/models"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	"github.com/lib/pq"
	"gopkg.in/gomail.v2"
)

var (
	dateFormat = "2006-01-02"
	s1         = `
	{
		"border": [
			{
			"type": "left",
			"color": "#000000",
			"style": 1
			},
			{
			"type": "top",
			"color": "#000000",
			"style": 1
			},
			{
			"type": "right",
			"color": "#000000",
			"style": 1
			},
			{
			"type": "bottom",
			"color": "#000000",
			"style": 1
			}
		],
		"fill": {
			"type": "pattern",
			"pattern": 1,
			"color": ["#96b753"]
		},
		"font": {
			"bold": true
		},
		"alignment": {
			"shrink_to_fit": true,
			"horizontal": "center"
		}
	}
	`
	s2 = `
	{
		"border": [
			{
			"type": "left",
			"color": "#000000",
			"style": 1
			},
			{
			"type": "top",
			"color": "#000000",
			"style": 1
			},
			{
			"type": "right",
			"color": "#000000",
			"style": 1
			},
			{
			"type": "bottom",
			"color": "#000000",
			"style": 1
			}
		],
		"fill": {
			"type": "pattern",
			"pattern": 1
		},
		"alignment": {
			"shrink_to_fit": true
		}
	}
	`
)

var genericOK = map[string]string{"message": "ok"}

type GenericResponse struct {
	Message string `json:"message"`
}

type API struct {
	Db    *sql.DB
	Redis *redis.Client
}

func NewAPI() *API {
	return &API{}
}

func sendError(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{
		"message": msg,
	})
}

func (api *API) GetTotal(query string, statement []interface{}) (total int32, err error) {
	err = api.Db.QueryRow(query, statement...).Scan(&total)
	return
}

func (api *API) BatchDeletes(c *gin.Context, table string) {
	u := ParsePayload(c)
	var req models.BatchDeleteRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	ids := req.Data
	if len(ids) == 0 {
		sendError(c, http.StatusBadRequest, "missing-data")
		return
	}

	needCheckCategories := table == "categories"
	needCheckProducts := table == "products"

	var errInvalid []models.RowError

	for i, id := range ids {
		if _, err := uuid.FromString(id); err != nil {
			errInvalid = append(errInvalid, models.RowError{
				Row:     i,
				Message: "invalid-id",
			})
			continue
		}

		if needCheckCategories {
			var exists bool
			if err := api.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM products WHERE category_id = $1 AND NOT deleted)", id).Scan(&exists); err != nil {
				sendError(c, http.StatusInternalServerError, err.Error())
				return
			}

			if exists {
				errInvalid = append(errInvalid, models.RowError{
					Row:     i,
					Message: "conflict-id",
				})
			}
		}

		if needCheckProducts {
			var exists bool
			if err := api.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM expenses WHERE product_id = $1 AND NOT deleted)", id).Scan(&exists); err != nil {
				sendError(c, http.StatusInternalServerError, err.Error())
				return
			}

			if exists {
				errInvalid = append(errInvalid, models.RowError{
					Row:     i,
					Message: "conflict-id",
				})
			}
		}
	}

	if len(errInvalid) > 0 {
		c.JSON(http.StatusBadRequest, models.RowResponseError{
			Message: "error",
			Detail:  errInvalid,
		})
		return
	}

	tx, err := api.Db.Begin()
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	defer tx.Rollback()
	var q string
	var stms = []interface{}{pq.Array(ids)}

	if u.Role == string(models.Customer) {
		q = " AND user_id = $2"
		stms = append(stms, u.Id)
	}

	tag, err := tx.Exec(`UPDATE `+table+` SET deleted = true WHERE id = ANY($1) AND NOT deleted`+q, stms...)
	if err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	t, _ := tag.RowsAffected()
	if int(t) != len(ids) {
		sendError(c, http.StatusNotFound, fmt.Sprintf("expected-%d-deleted-but-got-%d", len(ids), t))
		return
	}

	if err := tx.Commit(); err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, genericOK)
}

func (api *API) UpdatePassword(id, password string) (email string, err error) {
	err = api.Db.QueryRow(`UPDATE users SET password = crypt($1, gen_salt('bf', 8)) WHERE id = $2 AND NOT deleted RETURNING email`, password, id).Scan(&email)

	if err != nil {
		if err == sql.ErrNoRows {
			err = errors.New("not-found")
		}
		log.Println(err)
	}

	return
}

func sendEmailReset(email, token string) error {
	subject := os.Getenv("EMAIL_RESET_SUBJECT")
	emailSMTPPort := os.Getenv("EMAIL_SMTP_PORT")
	emailSMTPServer := os.Getenv("EMAIL_SMTP_SERVER")
	emailSMTPUsername := os.Getenv("EMAIL_SMTP_USERNAME")
	emailSMTPPassword := os.Getenv("EMAIL_SMTP_PASSWORD")
	emailFrom := os.Getenv("EMAIL_MESSAGE_FROM")

	f, err := os.Open("./templates/reset_password.html")
	if err != nil {
		log.Println(err)
		return err
	}

	body, err := ioutil.ReadAll(f)
	if err != nil {
		log.Println(err)
		return err
	}

	url := os.Getenv("WEB_URL") + "/forgot-password?token=" + token

	content := strings.ReplaceAll(string(body), "%URL%", url)

	log.Println(content)

	mailer := gomail.NewMessage()
	mailer.SetHeader("From", emailFrom)
	mailer.SetHeader("To", email)
	mailer.SetHeader("Subject", subject)
	mailer.SetBody("text/html", content)

	smtpPort, err := strconv.Atoi(emailSMTPPort)
	if err != nil {
		log.Println(err)
		return err
	}

	dialer := gomail.NewDialer(
		emailSMTPServer,
		smtpPort,
		emailSMTPUsername,
		emailSMTPPassword,
	)

	t := time.Now()
	err = dialer.DialAndSend(mailer)
	if err != nil {
		log.Println(err)
	}

	log.Println(time.Since(t))

	return err
}

func ParsePayload(c *gin.Context) (redis models.RedisPayload) {
	payload := c.Request.Header.Get("payload")

	err := json.Unmarshal([]byte(payload), &redis)
	if err != nil {
		log.Println(err)
	}

	return
}

func tokenGenerator() string {
	b := make([]byte, 32)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
