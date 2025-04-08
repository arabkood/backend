package handler

//
// import (
// 	"context"
//
// 	infraSqs "github.com/arabkood/api/infrastructure/sqs"
// 	appError "github.com/arabkood/api/internal/errors"
// 	"github.com/gin-gonic/gin"
// )
//
// type JobEndedRequest struct {
// 	Results       []byte
// 	Status        string
// 	RunnerVersion string
// }
//
// func JobEnded(c *gin.Context) {
// 	// 1. Remove the job from SQS
// 	// 2. Update submission in DB
// 	// 3. Delete user files from EFS
// 	// 4. Update DB tables (like giving user XP and marking modules as completed...)
// }
//
// func (h *RunnerHandler) Run(c *gin.Context) {
// 	job := &infraSqs.SubmissionJob{
// 		ID:     "hello-world",
// 		Type:   "test",
// 		Runner: "anything",
// 	}
//
// 	// Send the job
// 	err := h.submissionJobSender.Send(context.Background(), job)
// 	if err != nil {
// 		h.logger.Error().Str("handler", "runner.attempt").
// 			Err(err).
// 			Msg("Something went wrong with sending submission job to sqs")
// 		appError.ErrorInternal().AbortWithErrorJson(c)
// 		return
// 	}
// 	c.Status(200)
// }
