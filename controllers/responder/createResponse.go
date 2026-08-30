package controllers

import (
	"ccs-forms/db"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type textValidation struct {
	Required  bool    `json:"required"`
	Min       *int    `json:"min"`
	Max       *int    `json:"max"`
	MinLength *int    `json:"minLength"`
	MaxLength *int    `json:"maxLength"`
	Regex     *string `json:"regex"`
}

type answerInsert struct {
	QuestionID        int64
	Payload           json.RawMessage
	Structure         json.RawMessage
	CheckboxOptionIDs []int64
}

func validationMin(v textValidation) *int {
	if v.Min != nil {
		return v.Min
	}
	return v.MinLength
}

func validationMax(v textValidation) *int {
	if v.Max != nil {
		return v.Max
	}
	return v.MaxLength
}

func CreateResponse(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	formID, ok := publishedFormID(c)
	if !ok {
		return
	}

	var req CreateResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	if len(req.Answers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one answer is required"})
		return
	}

	ctx := c.Request.Context()

	var formStructure json.RawMessage
	var form ResponderForm
	err := db.Pool.QueryRow(ctx, `
		SELECT f.structure, f.deadline IS NOT NULL AND f.deadline < NOW()
		FROM published_forms f
		WHERE f.id = $1`,
		formID,
	).Scan(&formStructure, &form.Closed)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form"})
		return
	}
	if err := json.Unmarshal(formStructure, &form); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read form"})
		return
	}
	if form.Closed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Form is closed"})
		return
	}

	var existingResponseID int64
	err = db.Pool.QueryRow(ctx, `
		SELECT id
		FROM responses
		WHERE form_id = $1 AND user_id = $2`,
		formID, userID,
	).Scan(&existingResponseID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User has already responded to this form"})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not check existing response"})
		return
	}

	questionsByID := make(map[int64]FormQuestion)
	for _, section := range form.Sections {
		for _, question := range section.Questions {
			questionsByID[question.ID] = question
		}
	}
	if len(questionsByID) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Form has no questions"})
		return
	}

	answerRows := make([]answerInsert, 0, len(req.Answers))
	for questionIDKey, rawAnswer := range req.Answers {
		questionID, parseErr := strconv.ParseInt(questionIDKey, 10, 64)
		if parseErr != nil || questionID < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Answers must be keyed by valid question ids"})
			return
		}

		question, exists := questionsByID[questionID]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Answers do not match the form questions"})
			return
		}

		structure, marshalErr := json.Marshal(AnswerRequest{
			QuestionID: questionID,
			Payload:    rawAnswer,
		})
		if marshalErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not prepare answers"})
			return
		}

		var validation textValidation
		if len(question.Validation) > 0 {
			if err := json.Unmarshal(question.Validation, &validation); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read question validation"})
				return
			}
		}

		insert := answerInsert{
			QuestionID: questionID,
			Payload:    rawAnswer,
			Structure:  structure,
		}

		switch question.Type {
		case "text":
			var textAnswer string
			if err := json.Unmarshal(rawAnswer, &textAnswer); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Text answers must be strings"})
				return
			}
			if validation.Required && textAnswer == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Required text question is missing an answer"})
				return
			}
			if min := validationMin(validation); min != nil && len(textAnswer) < *min {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Text answer is shorter than the minimum length"})
				return
			}
			if max := validationMax(validation); max != nil && len(textAnswer) > *max {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Text answer exceeds the maximum length"})
				return
			}
			if validation.Regex != nil && *validation.Regex != "" {
				expr, err := regexp.Compile(*validation.Regex)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid question validation regex"})
					return
				}
				if !expr.MatchString(textAnswer) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Text answer does not match the required pattern"})
					return
				}
			}
		case "mcq":
			var optionIDs []int64
			if err := json.Unmarshal(rawAnswer, &optionIDs); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "MCQ answers must be arrays of option ids"})
				return
			}
			if validation.Required && len(optionIDs) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Required mcq question is missing an answer"})
				return
			}
			if len(optionIDs) != 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "MCQ answers must contain exactly one option id"})
				return
			}

			validOptions := make(map[int64]struct{}, len(question.Options))
			for _, option := range question.Options {
				validOptions[option.ID] = struct{}{}
			}
			if _, exists := validOptions[optionIDs[0]]; !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mcq option"})
				return
			}
		case "checkbox":
			var optionIDs []int64
			if err := json.Unmarshal(rawAnswer, &optionIDs); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Checkbox answers must be arrays of option ids"})
				return
			}
			if validation.Required && len(optionIDs) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Required checkbox question is missing an answer"})
				return
			}

			validOptions := make(map[int64]struct{}, len(question.Options))
			for _, option := range question.Options {
				validOptions[option.ID] = struct{}{}
			}
			for _, optionID := range optionIDs {
				if _, exists := validOptions[optionID]; !exists {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid checkbox option"})
					return
				}
			}
			insert.CheckboxOptionIDs = optionIDs
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported question type"})
			return
		}

		answerRows = append(answerRows, insert)
	}

	for questionID, question := range questionsByID {
		var validation textValidation
		if len(question.Validation) > 0 {
			if err := json.Unmarshal(question.Validation, &validation); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read question validation"})
				return
			}
		}
		if validation.Required {
			if _, exists := req.Answers[strconv.FormatInt(questionID, 10)]; !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Required questions must all be answered"})
				return
			}
		}
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}
	defer tx.Rollback(ctx)

	var response Response
	err = tx.QueryRow(ctx, `
		INSERT INTO responses (user_id, form_id)
		VALUES ($1, $2)
		RETURNING id, form_id, "timestamp"`,
		userID, formID,
	).Scan(&response.ID, &response.FormID, &response.Timestamp)
	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "User has already responded to this form"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}

	for _, answer := range answerRows {
		var answerID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO answers (response_id, question_id, payload, structure)
			VALUES ($1, $2, $3, $4)
			RETURNING id`,
			response.ID, answer.QuestionID, []byte(answer.Payload), []byte(answer.Structure),
		).Scan(&answerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, optionID := range answer.CheckboxOptionIDs {
			result, err := tx.Exec(ctx, `
				INSERT INTO checkbox_answers (checkbox_option_id, answer_id)
				SELECT o.id, $1
				FROM checkbox_options o
				WHERE o.id = $2 AND o.question_id = $3`,
				answerID, optionID, answer.QuestionID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if result.RowsAffected() == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid checkbox option"})
				return
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"response": response})
}
