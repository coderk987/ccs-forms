package routes

import (
	formControllers "ccs-forms/controllers/forms"
	responseControllers "ccs-forms/controllers/responses"
	"ccs-forms/middleware"

	"github.com/gin-gonic/gin"
)

func FormRoutes(router *gin.RouterGroup) {
	forms := router.Group("/")
	forms.Use(middleware.AuthMiddleware())

	draftForms := forms.Group("/draft_forms")
	draftForms.GET("/", formControllers.GetForms)
	draftForms.POST("/", formControllers.CreateDraftForm)
	draftForms.GET("/:id", formControllers.GetDraftForm)
	draftForms.PATCH("/:id", formControllers.UpdateDraftForm)
	draftForms.DELETE("/:id", formControllers.DeleteDraftForm)
	draftForms.PUT("/:id", formControllers.EditDraftForm)
	draftForms.POST("/:id/publish", formControllers.PublishDraftForm)

	publishedForms := forms.Group("/published_forms")
	publishedForms.GET("/", formControllers.GetPublishedForms)
	publishedForms.GET("/:id/responses", responseControllers.GetPublishedFormResponses)

	publishedForm := forms.Group("/published_form")
	publishedForm.PATCH("/:id", formControllers.UpdatePublishedForm)
	publishedForm.DELETE("/:id", formControllers.UnpublishForm)

	responderForms := forms.Group("/form")
	responderForms.GET("/:id", formControllers.GetResponderForm)
}
