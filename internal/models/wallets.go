package models

type Wallets struct {
	WalletID string `db:"wallet_id"`
	Balance  int    `db:"balance"`
}
