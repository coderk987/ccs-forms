package controllers

import (
	"ccs-forms/db"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type requestError struct {
	Status  int
	Message string
	Err     error
}

func (e *requestError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func writeRequestError(c *gin.Context, err error) bool {
	var reqErr *requestError
	if !errors.As(err, &reqErr) {
		return false
	}

	c.JSON(reqErr.Status, gin.H{"error": reqErr.Message})
	return true
}

type DraftForm struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
}

func draftFormID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form id must be a valid integer"})
		return 0, false
	}
	if id < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form id must be greater than zero"})
		return 0, false
	}
	return id, true
}

type DraftFormResponse struct {
	ID          int64
	Title       string
	Description string
	Sections    []FormSections
}

func createCompleteForm(c *gin.Context) (json.RawMessage, error) {
	userID := c.GetInt64("userID")
	form_id, ok := draftFormID(c)
	if !ok {
		return nil, &requestError{
			Status:  http.StatusBadRequest,
			Message: "Invalid form id",
		}
	}

	var form DraftFormResponse
	err := db.Pool.QueryRow(c.Request.Context(), `
		SELECT id, title, description
		FROM draft_forms
		WHERE id = $1 AND author_id = $2`, form_id, userID).Scan(&form.ID, &form.Title, &form.Description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &requestError{
				Status:  http.StatusNotFound,
				Message: "Draft form not found",
				Err:     err,
			}
		}
		return nil, &requestError{
			Status:  http.StatusInternalServerError,
			Message: "Could not fetch draft form",
			Err:     err,
		}
	}

	rows, err := db.Pool.Query(c.Request.Context(),
		`SELECT id, title, description, position FROM sections WHERE form_id = $1`,
		form_id,
	)
	if err != nil {
		return nil, &requestError{
			Status:  http.StatusInternalServerError,
			Message: "Could not fetch form sections",
			Err:     err,
		}
	}

	sections, err := pgx.CollectRows(rows, pgx.RowToStructByName[FormSections])
	rows.Close()
	if err != nil {
		return nil, &requestError{
			Status:  http.StatusInternalServerError,
			Message: "Could not read form sections",
			Err:     err,
		}
	}

	rows, err = db.Pool.Query(c.Request.Context(),
		`SELECT q.id, q.title, q.type, q.validation, q.position, q.section_id
		FROM questions q JOIN sections s ON q.section_id = s.id
		WHERE s.form_id = $1`,
		form_id,
	)
	if err != nil {
		return nil, &requestError{
			Status:  http.StatusInternalServerError,
			Message: "Could not fetch form questions",
			Err:     err,
		}
	}

	questions, err := pgx.CollectRows(rows, pgx.RowToStructByName[FormQuestions])
	rows.Close()
	if err != nil {
		return nil, &requestError{
			Status:  http.StatusInternalServerError,
			Message: "Could not read form questions",
			Err:     err,
		}
	}

	rows, err = db.Pool.Query(c.Request.Context(),
		`SELECT co.id, co.title, co.question_id
		FROM checkbox_options co
		JOIN questions q ON q.id = co.question_id
		JOIN sections s ON s.id = q.section_id
		WHERE s.form_id = $1

		UNION ALL

		SELECT mo.id, mo.title, mo.question_id
		FROM mcq_options mo
		JOIN questions q ON q.id = mo.question_id
		JOIN sections s ON s.id = q.section_id
		WHERE s.form_id = $1
		`,
		form_id,
	)
	if err != nil {
		return nil, &requestError{
			Status:  http.StatusInternalServerError,
			Message: "Could not fetch question options",
			Err:     err,
		}
	}

	options, err := pgx.CollectRows(rows, pgx.RowToStructByName[QuestionOptions])
	rows.Close()
	if err != nil {
		return nil, &requestError{
			Status:  http.StatusInternalServerError,
			Message: "Could not read question options",
			Err:     err,
		}
	}

	questionsMap := make(map[int64][]QuestionOptions)
	for _, option := range options {
		questionsMap[option.QuestionID] = append(questionsMap[option.QuestionID], option)
	}
	for i := range questions {
		questions[i].Options = questionsMap[questions[i].ID]
	}

	sectionsMap := make(map[int64][]FormQuestions)
	for _, question := range questions {
		sectionsMap[question.SectionID] = append(sectionsMap[question.SectionID], question)
	}
	for i := range sections {
		sections[i].Questions = sectionsMap[sections[i].ID]
	}

	form.Sections = sections

	jsonData, err := json.Marshal(form)
	if err != nil {
		return nil, &requestError{
			Status:  http.StatusInternalServerError,
			Message: "Could not serialize draft form",
			Err:     err,
		}
	}

	return jsonData, nil
}
