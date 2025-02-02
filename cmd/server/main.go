package main

import (
	"fmt"

	"github.com/foreground-eclipse/wallet/cmd/migrator"
	"github.com/foreground-eclipse/wallet/config"
	"github.com/foreground-eclipse/wallet/internal/handlers"
	"github.com/foreground-eclipse/wallet/internal/storage/postgres"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg := config.MustLoad("local")

	logger := setupLogger()
	if logger == nil {
		fmt.Println("Logger was not initialized")
	}

	err := migrator.Migrate(logger, cfg)
	if err != nil {
		logger.Error("Migrations were not applied", zap.Error(err))
	}
	storage, err := postgres.New(cfg)
	if err != nil {
		panic(err)
	}

	router := gin.Default()

	router.POST("/api/v1/wallet", handlers.HandleWalletOperation(logger, storage))
	router.GET("/api/v1/wallets/:walletId", handlers.HandleGetWalletBalance(logger, storage))

	router.Run(cfg.Server.Address)
}

func setupLogger() *zap.Logger {
	atomicLevel := zap.NewAtomicLevelAt(zap.InfoLevel)

	config := zap.Config{
		Level:            atomicLevel,
		Development:      true,
		Encoding:         "console",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := config.Build()

	if err != nil {
		fmt.Printf("Failed to create logger :%v\n", err)
		return nil
	}
	defer logger.Sync()

	if logger == nil {
		fmt.Println("Logger is nil!")
		return nil
	}
	logger.Info("Test info message")
	return logger
}
