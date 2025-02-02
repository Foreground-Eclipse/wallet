package models

import "time"

type Operation struct {
	ID        string    `db:"id"`
	WalletID  string    `db:"wallet_id"`
	Type      string    `db:"type"`
	Amount    int       `db:"amount"`
	Timestamp time.Time `db:"timestamp"`
}
