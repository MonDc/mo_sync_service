package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/mondc/mo_sync_service/internal/domain/user"
	"github.com/mondc/mo_sync_service/internal/infrastructure/persistence/mysql"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func initConfig() error {
	// 1. Load .env file first (if exists)
	if err := godotenv.Load("config/.env"); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// 2. Set up Viper for config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// 3. Read the config file
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	log.Printf("Config loaded from: %s", viper.ConfigFileUsed())

	// 4. Bind specific keys to environment variables
	viper.AutomaticEnv()

	viper.BindEnv("database.mi.host", "SQL_SERVER_22")
	viper.BindEnv("database.mi.user", "SQL_SERVER_22_DB_USER")
	viper.BindEnv("database.mi.password", "SQL_SERVER_22_DB_PASS")
	viper.BindEnv("database.mi.dbname", "MI")

	viper.BindEnv("database.mo.host", "SQL_SERVER_22")
	viper.BindEnv("database.mo.user", "SQL_SERVER_22_DB_USER")
	viper.BindEnv("database.mo.password", "SQL_SERVER_22_DB_PASS")
	viper.BindEnv("database.mo.dbname", "MO")

	return nil
}

func initLogger() (*zap.Logger, error) {
	logLevel := viper.GetString("log.level")

	var config zap.Config
	if logLevel == "development" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return config.Build()
}

func setupRouter(logger *zap.Logger, userSvc user.Service) *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"mo-user-sync"}`))
	})

	// POST /api/v1/users/{id}/sync
	router.HandleFunc("POST /api/v1/users/{id}/sync", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, `{"error":"user id is required"}`, http.StatusBadRequest)
			return
		}

		u, err := userSvc.SyncUser(r.Context(), id)
		if err != nil {
			logger.Error("sync failed", zap.Error(err), zap.String("user_id", id))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(u)
	})

	return router
}

func main() {
	// 1. Load configuration and bind environment variables
	if err := initConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Initialize logger
	logger, err := initLogger()
	if err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting MO User Sync Service")

	// 3. Initialize databases
	miDB, moDB, err := initDatabases(logger)
	if err != nil {
		logger.Fatal("Failed to connect to databases", zap.Error(err))
	}
	defer miDB.Close()
	defer moDB.Close()

	logger.Info("Both databases connected successfully")

	// 4. Initialize Repositories and Domain Service
	mainRepo := mysql.NewMainRepository(miDB, logger)
	localRepo := mysql.NewLocalRepository(moDB, logger)
	userSvc := user.NewDomainService(mainRepo, localRepo)

	// 5. Setup HTTP server
	router := setupRouter(logger, userSvc)
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", viper.GetString("server.port")),
		Handler:      router,
		ReadTimeout:  viper.GetDuration("server.read_timeout") * time.Second,
		WriteTimeout: viper.GetDuration("server.write_timeout") * time.Second,
		IdleTimeout:  viper.GetDuration("server.idle_timeout") * time.Second,
	}

	// 6. Start server in goroutine
	go func() {
		logger.Info("Server listening", zap.String("port", viper.GetString("server.port")))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed", zap.Error(err))
		}
	}()

	// 7. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

func initDatabases(logger *zap.Logger) (*sqlx.DB, *sqlx.DB, error) {
	// MI Database (main users)
	miDB, err := mysql.NewConnection(mysql.Config{
		Host:     viper.GetString("database.mi.host"),
		Port:     viper.GetInt("database.mi.port"),
		User:     viper.GetString("database.mi.user"),
		Password: viper.GetString("database.mi.password"),
		DBName:   viper.GetString("database.mi.dbname"),
		MaxConns: viper.GetInt("database.mi.max_conns"),
		MinConns: viper.GetInt("database.mi.min_conns"),
	}, logger.With(zap.String("db", "mi")))
	if err != nil {
		return nil, nil, fmt.Errorf("MI database: %w", err)
	}

	// MO Database (local users)
	moDB, err := mysql.NewConnection(mysql.Config{
		Host:     viper.GetString("database.mo.host"),
		Port:     viper.GetInt("database.mo.port"),
		User:     viper.GetString("database.mo.user"),
		Password: viper.GetString("database.mo.password"),
		DBName:   viper.GetString("database.mo.dbname"),
		MaxConns: viper.GetInt("database.mo.max_conns"),
		MinConns: viper.GetInt("database.mo.min_conns"),
	}, logger.With(zap.String("db", "mo")))
	if err != nil {
		return nil, nil, fmt.Errorf("MO database: %w", err)
	}

	return miDB, moDB, nil
}