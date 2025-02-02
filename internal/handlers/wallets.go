package handlers

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	requests "github.com/foreground-eclipse/wallet/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WalletBalanceGetter interface {
	GetWalletBalance(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error)
}

func HandleGetWalletBalance(logger *zap.Logger, balanceGetter WalletBalanceGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		const op = "api/v1/wallets/{walletId}"
		walletID := c.Param("walletId")

		logRequest(c, logger, "proceeding new request", zap.String("op", op), zap.String("walletId", walletID))

		if _, err := uuid.Parse(walletID); err != nil {
			logError(c, logger, errors.New("invalid wallet id format"), http.StatusBadRequest, "invalid walletId format")
			return
		}

		balanceChan := make(chan *requests.WalletBalanceResponse, 1)
		errChan := make(chan error, 1)
		go func() {
			balance, err := balanceGetter.GetWalletBalance(c.Request.Context(), walletID)
			if err != nil {
				errChan <- err
				return
			}
			balanceChan <- balance
		}()
		select {
		case balance := <-balanceChan:
			logRequest(c, logger, "request procceeded successfully", zap.String("walletId", walletID))
			c.JSON(http.StatusOK, requests.WalletOperationResponseOK(balance))
		case err := <-errChan:
			if errors.Is(err, sql.ErrNoRows) {
				logError(c, logger, errors.New("no such a wallet"), http.StatusNotFound, "wallet not found")
				return
			}
			logError(c, logger, err, http.StatusInternalServerError, "internal server error")
			return
		}
	}
}
func logRequest(c *gin.Context, logger *zap.Logger, message string, fields ...zap.Field) {
	logger.Info(message, fields...)
}
