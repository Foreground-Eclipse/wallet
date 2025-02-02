package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	requests "github.com/foreground-eclipse/wallet/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockWalletOperationHandler struct {
	ProcessOperationFunc func(ctx context.Context, req requests.WalletOperationRequest) error
	GetWalletBalanceFunc func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error)
}

func (m *mockWalletOperationHandler) ProcessOperation(ctx context.Context, req requests.WalletOperationRequest) error {
	if m.ProcessOperationFunc != nil {
		return m.ProcessOperationFunc(ctx, req)
	}
	return nil
}

func (m *mockWalletOperationHandler) GetWalletBalance(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
	if m.GetWalletBalanceFunc != nil {
		return m.GetWalletBalanceFunc(ctx, walletID)
	}
	return nil, nil
}

func TestHandleWalletOperation(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			panic(err)
		}
	}(logger)

	tests := []struct {
		name                 string
		requestBody          string
		mockProcessOperation func(ctx context.Context, req requests.WalletOperationRequest) error
		mockGetWalletBalance func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error)
		expectedStatus       int
		expectedBody         string
	}{
		{
			name:           "Invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"error","error":"failed to process request: invalid character 'i' looking for beginning of value"}`,
		},
		{
			name:           "Empty WalletId",
			requestBody:    `{"valletId": "", "operationType": "DEPOSIT", "amount": 1000}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"error","error":"bad request data: empty wallet id"}`,
		},
		{
			name:           "Invalid WalletId Format",
			requestBody:    `{"valletId": "invalid-uuid", "operationType": "DEPOSIT", "amount": 1000}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"error","error":"bad request data: invalid wallet id format"}`,
		},
		{
			name:           "Invalid operationType",
			requestBody:    `{"valletId": "a1b2c3d4-e5f6-7890-1234-567890abcdef", "operationType": "INVALID", "amount": 1000}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"error","error":"bad request data: operationType must be DEPOSIT or WITHDRAW"}`,
		},
		{
			name:           "Invalid amount",
			requestBody:    `{"valletId": "a1b2c3d4-e5f6-7890-1234-567890abcdef", "operationType": "DEPOSIT", "amount": -1000}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"error","error":"bad request data: amount must be a positive integer"}`,
		},
		{
			name:        "Success",
			requestBody: `{"valletId": "a1b2c3d4-e5f6-7890-1234-567890abcdef", "operationType": "DEPOSIT", "amount": 1000}`,
			mockProcessOperation: func(ctx context.Context, req requests.WalletOperationRequest) error {
				return nil
			},
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return &requests.WalletBalanceResponse{
					WalletID: "a1b2c3d4-e5f6-7890-1234-567890abcdef",
					Balance:  1000,
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK","data":{"walletId":"a1b2c3d4-e5f6-7890-1234-567890abcdef","operationType":"DEPOSIT","balance":1000}}`,
		},
		{
			name:        "Insufficient Funds Error",
			requestBody: `{"valletId": "a1b2c3d4-e5f6-7890-1234-567890abcdef", "operationType": "WITHDRAW", "amount": 1000}`,
			mockProcessOperation: func(ctx context.Context, req requests.WalletOperationRequest) error {
				return requests.InsufficientFundsError{}
			},
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return &requests.WalletBalanceResponse{
					WalletID: "a1b2c3d4-e5f6-7890-1234-567890abcdef",
					Balance:  1000,
				}, nil
			},
			expectedStatus: http.StatusForbidden,
			expectedBody:   `{"status":"error","error":"balance cant become negative: insufficient funds"}`,
		},
		{
			name:        "Wallet Not Found Error from ProcessOperation",
			requestBody: `{"valletId": "a1b2c3d4-e5f6-7890-1234-567890abcdef", "operationType": "WITHDRAW", "amount": 1000}`,
			mockProcessOperation: func(ctx context.Context, req requests.WalletOperationRequest) error {
				return sql.ErrNoRows
			},
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return &requests.WalletBalanceResponse{
					WalletID: "a1b2c3d4-e5f6-7890-1234-567890abcdef",
					Balance:  1000,
				}, nil
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":"error","error":"wallet not found: sql: no rows in result set"}`,
		},
		{
			name:        "Internal Server Error from ProcessOperation",
			requestBody: `{"valletId": "a1b2c3d4-e5f6-7890-1234-567890abcdef", "operationType": "WITHDRAW", "amount": 1000}`,
			mockProcessOperation: func(ctx context.Context, req requests.WalletOperationRequest) error {
				return errors.New("some other error")
			},
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return &requests.WalletBalanceResponse{
					WalletID: "a1b2c3d4-e5f6-7890-1234-567890abcdef",
					Balance:  1000,
				}, nil
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"error","error":"internal server error: some other error"}`,
		},
		{
			name:        "Wallet Not Found Error from GetWalletBalance",
			requestBody: `{"valletId": "a1b2c3d4-e5f6-7890-1234-567890abcdef", "operationType": "WITHDRAW", "amount": 1000}`,
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return nil, sql.ErrNoRows
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":"error","error":"wallet not found: sql: no rows in result set"}`,
		},
		{
			name:        "Internal Server Error from GetWalletBalance",
			requestBody: `{"valletId": "a1b2c3d4-e5f6-7890-1234-567890abcdef", "operationType": "WITHDRAW", "amount": 1000}`,
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return nil, errors.New("some other error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"status":"error","error":"internal server error: some other error"}`,
		},
		{
			name:        "Success GetWalletBalance",
			requestBody: `{"valletId": "a1b2c3d4-e5f6-7890-1234-567890abcdef", "operationType": "WITHDRAW", "amount": 1000}`,
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return &requests.WalletBalanceResponse{
					WalletID: "a1b2c3d4-e5f6-7890-1234-567890abcdef",
					Balance:  1000,
				}, nil
			},
			mockProcessOperation: func(ctx context.Context, req requests.WalletOperationRequest) error {
				return nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK","data":{"walletId":"a1b2c3d4-e5f6-7890-1234-567890abcdef","operationType":"WITHDRAW","balance":1000}}`,
		},
		{
			name:        "Empty JSON",
			requestBody: "{}",
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return &requests.WalletBalanceResponse{
					WalletID: "a1b2c3d4-e5f6-7890-1234-567890abcdef",
					Balance:  1000,
				}, nil
			},
			mockProcessOperation: func(ctx context.Context, req requests.WalletOperationRequest) error {
				return nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"error","error":"bad request data: empty wallet id"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := &mockWalletOperationHandler{
				ProcessOperationFunc: tt.mockProcessOperation,
				GetWalletBalanceFunc: tt.mockGetWalletBalance,
			}
			recorder := httptest.NewRecorder()
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(recorder)
			c.Request, _ = http.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewBufferString(tt.requestBody))
			c.Request.Header.Set("Content-Type", "application/json")

			HandleWalletOperation(logger, mockHandler)(c)
			assert.Equal(t, tt.expectedStatus, recorder.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, recorder.Body.String())
			}
		})
	}
}
