package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"

	requests "github.com/foreground-eclipse/wallet/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WalletOperationHandler interface {
	ProcessOperation(ctx context.Context, req requests.WalletOperationRequest) error
	GetWalletBalance(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error)
}

func HandleWalletOperation(logger *zap.Logger, handler WalletOperationHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req requests.WalletOperationRequest
		const op = "api/v1/wallet"

		logger.Info("proceeding new request", zap.String("op", op))

		if err := c.BindJSON(&req); err != nil {
			if errors.Is(err, io.EOF) {
				logError(c, logger, errors.New("empty json"), http.StatusBadRequest, "failed to process request")
			}
			logError(c, logger, err, http.StatusBadRequest, "failed to process request")
			return
		}

		reqBody, err := json.Marshal(req)
		if err != nil {
			logError(c, logger, err, http.StatusBadRequest, "failed to marshal request body")
			return
		}
		logger.Info("request data",
			zap.String("method", c.Request.Method),
			zap.String("URL", c.Request.URL.String()),
			zap.String("body", string(reqBody)),
		)

		validationError := validateRequest(req)
		if validationError != nil {
			logError(c, logger, validationError, http.StatusBadRequest, "bad request data")
			return
		}
		var wg sync.WaitGroup
		errChan := make(chan error, 1)
		balanceChan := make(chan *requests.WalletBalanceResponse, 1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			err = handler.ProcessOperation(c.Request.Context(), req)
			if err != nil {
				errChan <- err
				return
			}
			walletBalance, err := handler.GetWalletBalance(c, req.WalletID)
			if err != nil {
				errChan <- err
				return
			}
			balanceChan <- walletBalance
		}()
		go func() {
			wg.Wait()
			close(errChan)
			close(balanceChan)
		}()
		select {
		case balance := <-balanceChan:
			logger.Info("request procceeded successfully", zap.String("request_body", string(reqBody)))
			c.JSON(http.StatusOK, requests.WalletOperationResponseOK(map[string]interface{}{
				"walletId":      req.WalletID,
				"operationType": req.OperationType,
				"balance":       balance.Balance,
			}))
		case err := <-errChan:
			if errors.Is(err, sql.ErrNoRows) {
				logError(c, logger, err, http.StatusNotFound, "wallet not found")
				return
			}
			var insufficientFundsErr requests.InsufficientFundsError
			if errors.As(err, &insufficientFundsErr) {
				logError(c, logger, err, http.StatusForbidden, "balance cant become negative")
				return
			}
			logError(c, logger, err, http.StatusInternalServerError, "internal server error")
			return
		}
	}
}

func validateRequest(req requests.WalletOperationRequest) error {
	if req.WalletID == "" {
		return errors.New("empty wallet id")
	}
	if _, err := uuid.Parse(req.WalletID); err != nil {
		return errors.New("invalid wallet id format")
	}

	allowedOperations := map[string]bool{
		"DEPOSIT":  true,
		"WITHDRAW": true,
	}
	if !allowedOperations[req.OperationType] {

		return errors.New("operationType must be DEPOSIT or WITHDRAW")
	}
	if req.Amount <= 0 {
		return errors.New("amount must be a positive integer")
	}
	return nil
}
func logError(c *gin.Context, logger *zap.Logger, err error, status int, message string) {
	logger.Warn(message, zap.Error(err))
	c.JSON(status, requests.Error(errors.New(message+": "+err.Error())))
}
