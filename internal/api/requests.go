package requests

const (
	StatusOK    = "OK"
	StatusError = "error"
)

type WalletOperationRequest struct {
	WalletID      string `json:"valletId"`
	OperationType string `json:"operationType"` // "DEPOSIT" or "WITHDRAW"
	Amount        int    `json:"amount"`
}

type WalletOperationResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
	Data   interface{} `json:"data,omitempty"`
}

type WalletBalanceResponse struct {
	WalletID string `json:"walletId"`
	Balance  int    `json:"balance"`
}

func WalletOperationResponseOK(data interface{}) WalletOperationResponse {
	return WalletOperationResponse{
		Status: StatusOK,
		Data:   data,
	}
}

func Error(err error) WalletOperationResponse {
	return WalletOperationResponse{
		Status: StatusError,
		Error:  err.Error(),
	}
}
