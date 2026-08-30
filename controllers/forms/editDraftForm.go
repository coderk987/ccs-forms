package controllers

import (
	"ccs-forms/db"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type EditDraftFormRequest struct {
	Sections []FormSections `json:"sections"`
}

type FormSections struct {
	ID          int64           `json:"id" db:"id"`
	Title       string          `json:"title" db:"title"`
	Description string          `json:"description" db:"description"`
	Position    int64           `json:"position" db:"position"`
	Status      string          `json:"status" db:"-"`
	Questions   []FormQuestions `json:"questions" db:"-"`
}

type FormQuestions struct {
	ID         int64             `json:"id" db:"id"`
	Title      string            `json:"title" db:"title"`
	Type       string            `json:"type" db:"type"`
	Validation json.RawMessage   `json:"validation" db:"validation"`
	Position   int64             `json:"position" db:"position"`
	Status     string            `json:"status" db:"-"`
	Options    []QuestionOptions `json:"options" db:"-"`
	SectionID  int64             `json:"section_id" db:"section_id"`
}

type QuestionOptions struct {
	ID         int64  `json:"id" db:"id"`
	Status     string `json:"status" db:"-"`
	Title      string `json:"title" db:"title"`
	QuestionID int64  `json:"question_id" db:"question_id"`
}

func EditDraftForm(c *gin.Context) {
	userID := c.GetInt64("userID")
	id, ok := draftFormID(c)
	if !ok {
		return
	}

	var row int64
	err := db.Pool.QueryRow(c.Request.Context(),
		`SELECT author_id FROM draft_forms WHERE id=$1`,
		id,
	).Scan(&row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Draft form not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
		return
	}
	if row != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not the Owner of the form"})
		return
	}

	var req EditDraftFormRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Body", "message": err.Error()})
		return
	}

	//Will add Retry later on
	for sectionIndex := range req.Sections {
		section := &req.Sections[sectionIndex]
		if section.Status == "UNCHANGED" {
			continue
		}

		//sql queries for different actions
		if section.Status == "ADD" {
			err := db.Pool.QueryRow(c.Request.Context(),
				`INSERT INTO sections (title, description, position, form_id)
				 VALUES ($1, $2, $3, $4)
				 RETURNING id`,
				section.Title, section.Description, section.Position, id,
			).Scan(&section.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
				return
			}
		} else if section.Status == "CHANGE" {
			_, err := db.Pool.Exec(c.Request.Context(),
				`UPDATE sections SET title = $1, description = $2, position = $3 WHERE id = $4 AND form_id = $5`,
				section.Title, section.Description, section.Position, section.ID, id,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
				return
			}
		} else if section.Status == "DELETE" {
			_, err := db.Pool.Exec(c.Request.Context(),
				`DELETE FROM sections WHERE id = $1 AND form_id = $2`,
				section.ID, id,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
				return
			}
			continue
		}

		//iterating over questions
		for questionIndex := range section.Questions {
			question := &section.Questions[questionIndex]
			if question.Status == "UNCHANGED" {
				continue
			}

			//sql queries for actions on different questions
			if question.Status == "ADD" {
				err := db.Pool.QueryRow(c.Request.Context(),
					`INSERT INTO questions (title, position, type, validation, section_id)
					 VALUES ($1, $2, $3, $4, $5)
					 RETURNING id`,
					question.Title, question.Position, question.Type, question.Validation, section.ID,
				).Scan(&question.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
					return
				}
			} else if question.Status == "CHANGE" {
				_, err := db.Pool.Exec(c.Request.Context(),
					`UPDATE questions SET title = $1, position = $2, type = $3, validation = $4 WHERE id = $5`,
					question.Title, question.Position, question.Type, question.Validation, question.ID,
				)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
					return
				}
			} else if question.Status == "DELETE" {
				_, err := db.Pool.Exec(c.Request.Context(),
					`DELETE FROM questions WHERE id = $1 AND section_id = $2`,
					question.ID, section.ID,
				)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
					return
				}
				continue
			}

			//checking for options
			if question.Type == "checkbox" || question.Type == "mcq" {
				//validating and making sure no sql injection stuff
				allowedTables := map[string]string{
					"mcq":      "mcq_options",
					"checkbox": "checkbox_options",
				}

				tableName, ok := allowedTables[question.Type]
				if !ok {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Bad Request"})
					return
				}

				for optionIndex := range question.Options {
					option := &question.Options[optionIndex]
					if option.Status == "UNCHANGED" {
						continue
					}
					//sql querie sfor diff actions on options
					if option.Status == "ADD" {
						err := db.Pool.QueryRow(c.Request.Context(),
							`INSERT INTO `+tableName+` (title, question_id)
							 VALUES ($1, $2)
							 RETURNING id`,
							option.Title, question.ID,
						).Scan(&option.ID)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
							return
						}
					} else if option.Status == "CHANGE" {
						_, err := db.Pool.Exec(c.Request.Context(),
							`UPDATE `+tableName+` SET title = $1 WHERE id = $2`,
							option.Title, option.ID,
						)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
							return
						}
					} else if question.Status == "DELETE" {
						_, err := db.Pool.Exec(c.Request.Context(),
							`DELETE FROM `+tableName+` WHERE id = $1 AND question_id = $2`,
							option.ID, question.ID,
						)
						if err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update properly"})
							return
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Saved Properly"})
}
