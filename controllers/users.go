package controllers

import (
	"budgetingapi/models"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

// only customer
func (api *API) Register(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := validateUser(user, true); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	var exists bool
	if err := api.Db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND NOT deleted)", user.Email).Scan(&exists); err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if exists {
		sendError(c, http.StatusConflict, "email-already-exist")
		return
	}

	if _, err := api.Db.Exec(`
		INSERT INTO users (email, name, password, role, created_at, updated_at)
		VALUES ($1, $2, crypt($3, gen_salt('bf', 8)), 'CUSTOMER', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, user.Email, user.Name, user.Password); err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, genericOK)
}

func (api *API) GetUser(c *gin.Context) {
	userId := ParsePayload(c).Id

	if userId == "" {
		sendError(c, http.StatusBadRequest, "missing-id")
		return
	}

	if _, err := uuid.FromString(userId); err != nil {
		sendError(c, http.StatusBadRequest, "invalid-id")
		return
	}

	var user models.User

	if err := api.Db.QueryRow("SELECT id, email, name, role, created_at, updated_at FROM users WHERE id = $1 AND NOT deleted", userId).
		Scan(&user.Id, &user.Email, &user.Name, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			sendError(c, http.StatusNotFound, "user-not-found")
			return
		}

		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, user)
}

func (api *API) UpdateUser(c *gin.Context) {
	userId := ParsePayload(c).Id

	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	updatePassword := user.Password != ""

	if err := validateUser(user, updatePassword); err != nil {
		log.Println(err)
		sendError(c, http.StatusBadRequest, err.Error())
		return
	}

	q := "UPDATE users SET name = $1, email = $2"
	stms := []interface{}{user.Name, user.Email}

	if updatePassword {
		q += " password = crypt($3, gen_salt('bf', 8))"
		stms = append(stms, user.Password)
	}

	stms = append(stms, userId)
	q += fmt.Sprintf(" WHERE id = $%d AND NOT deleted", len(stms))

	if _, err := api.Db.Exec(q, stms...); err != nil {
		log.Println(err)
		sendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, genericOK)
}

func validateUser(user models.User, checkPassword bool) error {
	if user.Email == "" {
		return errors.New("missing-email")
	}

	if user.Name == "" {
		return errors.New("missing-name")
	}

	if _, err := mail.ParseAddress(user.Email); err != nil {
		log.Println(err)
		return errors.New("invalid-email")
	}

	if checkPassword {
		if user.Password == "" {
			return errors.New("missing-password")
		}

		if len(user.Password) < 8 {
			return errors.New("password-must-be-at-least-8-characters")
		}
	}

	return nil
}
