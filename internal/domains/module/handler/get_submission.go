package handler

import (
	"context"
	"net/http"
	"time"

	submissionRepo "github.com/arabkood/backend/internal/domains/module/repo/submission"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *ModuleHandler) GetSubmission(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get user ID from auth context
	userID, exists := c.Get("userID")
	if !exists {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}
	userId := userID.(uuid.UUID).String()
	if userId == "" {
		appError.ErrorUnauthorized().AbortWithErrorJson(c)
		return
	}

	// Get submission ID from path
	moduleID := c.Param("id")
	if moduleID == "" {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	submissionRepo := submissionRepo.NewSubmissionRepository(h.db)

	submission, aerr := submissionRepo.GetSubmissionByUserModule(ctx, userId, moduleID)
	if aerr != nil {
		aerr.Log(h.logger.Error().Str("handler", "module.getSubmission"), true).
			Str("userId", userId).
			Str("moduleID", moduleID).
			Msg("Failed to get submission")
		aerr.AbortWithErrorJson(c)
		return
	}

	c.JSON(http.StatusOK, submission)
}
