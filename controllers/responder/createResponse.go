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
	CheckboxOptionIDs []int64
}

type answerBatchInsert struct {
	QuestionID int64           `json:"question_id"`
	Payload    json.RawMessage `json:"payload"`
}

type checkboxAnswerInsert struct {
	AnswerID int64
	OptionID int64
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
			seenOptions := make(map[int64]struct{}, len(optionIDs))
			for _, optionID := range optionIDs {
				if _, exists := validOptions[optionID]; !exists {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid checkbox option"})
					return
				}
				if _, exists := seenOptions[optionID]; exists {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Checkbox answers cannot contain duplicate options"})
					return
				}
				seenOptions[optionID] = struct{}{}
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
	responseStructure, err := json.Marshal(req.Answers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not prepare response"})
		return
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}
	defer tx.Rollback(ctx)

	var response Response
	err = tx.QueryRow(ctx, `
		INSERT INTO responses (user_id, form_id, structure)
		VALUES ($1, $2, $3)
		RETURNING id, form_id, "timestamp", structure`,
		userID, formID, responseStructure,
	).Scan(&response.ID, &response.FormID, &response.Timestamp, &response.Structure)
	if err != nil {
		if isUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "User has already responded to this form"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}

	batch := make([]answerBatchInsert, 0, len(answerRows))
	for _, answer := range answerRows {
		batch = append(batch, answerBatchInsert{
			QuestionID: answer.QuestionID,
			Payload:    answer.Payload,
		})
	}
	batchJSON, err := json.Marshal(batch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not prepare answers"})
		return
	}

	answerIDs := make(map[int64]int64, len(batch))
	rows, err := tx.Query(ctx, `
		INSERT INTO answers (response_id, question_id, payload)
		SELECT $1, x.question_id, x.payload
		FROM jsonb_to_recordset($2::jsonb) AS x(
			question_id BIGINT,
			payload JSONB
		)
		RETURNING id, question_id`, response.ID, batchJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save answers"})
		return
	}
	for rows.Next() {
		var answerID, questionID int64
		if err := rows.Scan(&answerID, &questionID); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read saved answers"})
			return
		}
		answerIDs[questionID] = answerID
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save answers"})
		return
	}
	rows.Close()

	checkboxPairs := make([]checkboxAnswerInsert, 0)
	for _, answer := range answerRows {
		answerID := answerIDs[answer.QuestionID]
		for _, optionID := range answer.CheckboxOptionIDs {
			checkboxPairs = append(checkboxPairs, checkboxAnswerInsert{AnswerID: answerID, OptionID: optionID})
		}
	}
	if len(checkboxPairs) > 0 {
		answerIDValues := make([]int64, 0, len(checkboxPairs))
		optionIDValues := make([]int64, 0, len(checkboxPairs))
		for _, pair := range checkboxPairs {
			answerIDValues = append(answerIDValues, pair.AnswerID)
			optionIDValues = append(optionIDValues, pair.OptionID)
		}
		result, err := tx.Exec(ctx, `
			INSERT INTO checkbox_answers (checkbox_option_id, answer_id)
			SELECT o.id, x.answer_id
			FROM unnest($1::bigint[], $2::bigint[]) AS x(answer_id, option_id)
			JOIN checkbox_options o ON o.id = x.option_id
			JOIN answers a ON a.id = x.answer_id
				AND a.question_id = o.question_id`, answerIDValues, optionIDValues)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save checkbox answers"})
			return
		}
		if result.RowsAffected() != int64(len(checkboxPairs)) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid checkbox option"})
			return
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"response": response})
}
