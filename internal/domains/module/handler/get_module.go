package handler

import (
	"context"
	"time"

	"github.com/arabkood/backend/internal/domains/module/interfaces/exercise"
	"github.com/arabkood/backend/internal/domains/module/interfaces/module"
	exerciseRepo "github.com/arabkood/backend/internal/domains/module/repo/exercise"
	repo "github.com/arabkood/backend/internal/domains/module/repo/module"
	appError "github.com/arabkood/backend/pkg/errors"
	"github.com/gin-gonic/gin"
)

type GetModuleResponse struct {
	Module   *module.Module     `json:"module"`
	Exercise *exercise.Exercise `json:"exercise,omitempty"`
}

func (h *ModuleHandler) GetModule(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	slug := c.Param("slug")
	if slug == "" {
		appError.ErrorInvalidInput().AbortWithErrorJson(c)
		return
	}

	// Create repository instance
	moduleRepo := repo.NewModuleRepository(h.db)

	// Get module
	module, aerr := moduleRepo.GetModuleBySlug(ctx, slug)

	if aerr != nil {
		if aerr.Type == appError.ErrNotFound {
			appError.ErrorModuleNotFound().AbortWithErrorJson(c)
			return
		}
		aerr.Log(h.logger.Error().Str("handler", "module.GetModule"), true).
			Msg("Failed to get module")
		appError.ErrorInternal().AbortWithErrorJson(c)
		return
	}

	res := GetModuleResponse{
		Module: module,
	}

	// Get Module Content from S3
	if module.Type == "exercise" {
		exr, err := exerciseRepo.GetExercise(c, h.s3Client, h.config.Aws.ExercisesBucketName, module.Source)
		// fmt.Println(exr, err)
		if err != nil {
			h.logger.Error().Err(err).Str("handler", "module.GetModule").
				Msg("Failed to get module")
			appError.ErrorInternal().AbortWithErrorJson(c)
			return
		}
		res.Exercise = exr
	}

	// Return response
	c.JSON(200, res)
}
