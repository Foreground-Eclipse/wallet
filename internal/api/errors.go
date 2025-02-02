package requests

type InsufficientFundsError struct{}

func (e InsufficientFundsError) Error() string {
	return "insufficient funds"
}
