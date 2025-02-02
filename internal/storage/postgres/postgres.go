package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/foreground-eclipse/wallet/config"
	requests "github.com/foreground-eclipse/wallet/internal/api"
	"github.com/foreground-eclipse/wallet/internal/models"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// docker run --name walletDB -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=Tatsh -e POSTGRES_DB=wallet -d postgres
type Storage struct {
	db *sql.DB
}

func genUUID() string {
	return uuid.New().String()
}

func New(cfg *config.Config) (*Storage, error) {
	const op = "storage.postgres.New"

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.SSLMode)
	db, err := sql.Open("postgres", connStr)

	if err != nil {
		return nil, fmt.Errorf("%s : %w", op, err)
	}
	db.SetMaxOpenConns(250)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(time.Minute * 1)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{
		db: db,
	}, nil
}

func (s *Storage) CreateWallet(ctx context.Context, walletID string) error {
	op := "database.CreateWallet"
	_, err := s.db.ExecContext(ctx, "INSERT INTO wallets (wallet_id) VALUES ($1)", walletID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func (s *Storage) GetWallet(ctx context.Context, walletID string) (*models.Wallets, error) {
	op := "database.GetWallet"
	var wallet models.Wallets
	err := s.db.QueryRowContext(ctx, `SELECT wallet_id, balance FROM wallets WHERE wallet_id = $1`, walletID).Scan(&wallet.WalletID, &wallet.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: wallet with id %s not found: %w", op, walletID, err)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &wallet, nil
}

func (s *Storage) ProcessOperation(ctx context.Context, req requests.WalletOperationRequest) error {
	op := "database.ProcessOperation"

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				fmt.Printf("transaction rollback failed: %v", rbErr)
			}
		}
	}()

	wallet, err := s.GetWallet(ctx, req.WalletID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err = s.CreateWallet(ctx, req.WalletID); err != nil {
				return fmt.Errorf("%s: create wallet error: %w", op, err)
			}
		} else {
			return fmt.Errorf("%s: get wallet error: %w", op, err)
		}

		wallet = &models.Wallets{
			WalletID: req.WalletID,
			Balance:  0,
		}
	}
	var newBalance int
	switch req.OperationType {
	case "DEPOSIT":
		newBalance = wallet.Balance + req.Amount
	case "WITHDRAW":
		if wallet.Balance < req.Amount {
			return requests.InsufficientFundsError{}
		}
		newBalance = wallet.Balance - req.Amount
	default:
		return fmt.Errorf("%s: invalid operation type", op)
	}
	_, err = tx.ExecContext(ctx, "UPDATE wallets SET balance = $1 WHERE wallet_id = $2", newBalance, req.WalletID)
	if err != nil {
		return fmt.Errorf("%s: update wallet error: %w", op, err)
	}
	operation := models.Operation{
		ID:        genUUID(),
		WalletID:  req.WalletID,
		Type:      req.OperationType,
		Amount:    req.Amount,
		Timestamp: time.Now(),
	}
	_, err = tx.ExecContext(ctx, `
    INSERT INTO operations (id, wallet_id, type, amount, timestamp)
    VALUES ($1, $2, $3, $4, $5)
    `, operation.ID, operation.WalletID, operation.Type, operation.Amount, operation.Timestamp)
	if err != nil {
		return fmt.Errorf("%s: insert operation error: %w", op, err)
	}

	err = tx.Commit()

	return err

}

func (s *Storage) GetWalletBalance(ctx context.Context, walletID string) (*requests.WalletBalanceResponse, error) {
	op := "database.GetWalletBalance"
	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: wallet with id %s not found: %w", op, walletID, err)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return &requests.WalletBalanceResponse{
		WalletID: wallet.WalletID,
		Balance:  wallet.Balance,
	}, nil
}
