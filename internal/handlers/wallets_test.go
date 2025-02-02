package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	requests "github.com/foreground-eclipse/wallet/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockWalletBalanceGetter struct {
	GetWalletBalanceFunc func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error)
}

func (m *mockWalletBalanceGetter) GetWalletBalance(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
	fmt.Println("mock GetWalletBalance called with walletID:", walletID)
	if m.GetWalletBalanceFunc != nil {
		balance, err := m.GetWalletBalanceFunc(ctx, walletID)
		fmt.Printf("Mock returned balance: %v, error: %v\n", balance, err)
		return balance, err
	}
	fmt.Println("mock GetWalletBalance return nil, nil")
	return nil, nil
}

func TestHandleGetWalletBalance(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			panic(err)
		}
	}(logger)
	tests := []struct {
		name                 string
		walletId             string
		mockGetWalletBalance func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error)
		expectedStatus       int
		expectedBody         string
	}{
		{
			name:           "Invalid UUID",
			walletId:       "invalid-uuid",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"status":"error","error":"invalid walletId format: invalid wallet id format"}`,
		},
		{
			name:     "Success",
			walletId: "a1b2c3d4-e5f6-7890-1234-567890abcdef",
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return &requests.WalletBalanceResponse{
					WalletID: "a1b2c3d4-e5f6-7890-1234-567890abcdef",
					Balance:  6000,
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"OK","data":{"walletId":"a1b2c3d4-e5f6-7890-1234-567890abcdef","balance":6000}}`,
		},
		{
			name:     "Wallet Not Found",
			walletId: "a887e82a-433b-4484-b6ec-820d6451c8bd",
			mockGetWalletBalance: func(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
				return nil, sql.ErrNoRows
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   `{"status":"error","error":"wallet not found: no such a wallet"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockHandler := &mockWalletBalanceGetter{
				GetWalletBalanceFunc: tt.mockGetWalletBalance,
			}
			recorder := httptest.NewRecorder()
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(recorder)
			c.Request, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/wallets/%s", tt.walletId), bytes.NewBufferString(""))
			c.Params = []gin.Param{{Key: "walletId", Value: tt.walletId}}
			c.Request.Header.Set("Content-Type", "application/json")

			HandleGetWalletBalance(logger, mockHandler)(c)
			assert.Equal(t, tt.expectedStatus, recorder.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, recorder.Body.String())
			}

		})
	}
}
