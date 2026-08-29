package controllers

import (
	"ccs-forms/db"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetForm(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	formID, ok := publishedFormID(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()

	// only a live form the responder is allowed to see
	form := ResponderForm{Sections: []FormSection{}}
	err := db.Pool.QueryRow(ctx, `
		SELECT f.id, f.title, f.description, f.deadline
		FROM published_forms f
		WHERE f.id = $1
			AND (f.author_id = $2 OR EXISTS (
				SELECT 1 FROM view_permissions v
				WHERE v.form_id = f.id AND v.user_id = $2))`, formID, userID).Scan(
		&form.ID, &form.Title, &form.Description, &form.Deadline,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form"})
		return
	}
	form.Closed = form.Deadline != nil && form.Deadline.Before(time.Now())

	sectionRows, err := db.Pool.Query(ctx, `
		SELECT id, title, description
		FROM sections
		WHERE form_id = $1
		ORDER BY position, id`, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form sections"})
		return
	}
	defer sectionRows.Close()

	sections := map[int64]*FormSection{}
	for sectionRows.Next() {
		var section FormSection
		section.Questions = []FormQuestion{}
		if err := sectionRows.Scan(&section.ID, &section.Title, &section.Description); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form sections"})
			return
		}
		form.Sections = append(form.Sections, section)
	}
	if sectionRows.Err() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form sections"})
		return
	}
	for i := range form.Sections {
		sections[form.Sections[i].ID] = &form.Sections[i]
	}

	questionRows, err := db.Pool.Query(ctx, `
		SELECT q.id, q.title, q.type, q.validation, q.section_id
		FROM questions q
		JOIN sections s ON s.id = q.section_id
		WHERE s.form_id = $1
		ORDER BY q.position, q.id`, formID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form questions"})
		return
	}
	defer questionRows.Close()

	questionIDs := []int64{}
	for questionRows.Next() {
		var question FormQuestion
		var sectionID int64
		question.Options = []FormOption{}
		if err := questionRows.Scan(
			&question.ID, &question.Title, &question.Type, &question.Validation, &sectionID,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form questions"})
			return
		}
		section, ok := sections[sectionID]
		if !ok {
			continue
		}
		section.Questions = append(section.Questions, question)
		questionIDs = append(questionIDs, question.ID)
	}
	if questionRows.Err() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form questions"})
		return
	}

	if len(questionIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"form": form})
		return
	}

	questions := map[int64]*FormQuestion{}
	for i := range form.Sections {
		for j := range form.Sections[i].Questions {
			question := &form.Sections[i].Questions[j]
			questions[question.ID] = question
		}
	}

	optionRows, err := db.Pool.Query(ctx, `
		SELECT id, question_id, title, NULL::bigint AS unlock_section_id
		FROM checkbox_options
		WHERE question_id = ANY($1)
		UNION ALL
		SELECT id, question_id, title, unlock_section_id
		FROM mcq_options
		WHERE question_id = ANY($1)
		ORDER BY question_id, id`, questionIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form options"})
		return
	}
	defer optionRows.Close()

	for optionRows.Next() {
		var option FormOption
		var questionID int64
		if err := optionRows.Scan(&option.ID, &questionID, &option.Title, &option.UnlockSectionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form options"})
			return
		}
		if question, ok := questions[questionID]; ok {
			question.Options = append(question.Options, option)
		}
	}
	if optionRows.Err() != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get form options"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"form": form})
}
