package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func ParseID(c *gin.Context, rawID string, log *zap.Logger) (uuid.UUID, bool) {
	id, err := uuid.Parse(rawID)
	if err != nil {
		if log != nil {
			log.Warn("invalid uuid in path", zap.String("raw_id", rawID), zap.Error(err))
		}
		return uuid.Nil, false
	}
	return id, true
}
