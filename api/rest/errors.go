package rest

import (
	"errors"
	"net/http"

	svcerrors "admin-service/pkg/errors"

	"github.com/gin-gonic/gin"
)

type errorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondWithError(c *gin.Context, err error) {
	if svcErr := findServiceError(err); svcErr != nil {
		c.JSON(svcErr.Status(), errorPayload{
			Code:    svcErr.Code(),
			Message: svcErr.Message(),
		})
		return
	}

	c.JSON(http.StatusInternalServerError, errorPayload{
		Code:    "unknown_error",
		Message: "unexpected error occurred",
	})
}

func findServiceError(err error) svcerrors.ServiceError {
	var svcErr svcerrors.ServiceError
	if errors.As(err, &svcErr) {
		return svcErr
	}
	return nil
}
