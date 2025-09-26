package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"seer/config"
	"seer/internal/chat"
	"seer/internal/geo"
	"seer/internal/handlers"
	"seer/internal/market"
	"seer/internal/middlewares"
	"seer/internal/repos"
	"seer/internal/utils"
	"seer/internal/ws"
	"strings"
	"time"

	_ "go.uber.org/automaxprocs"

	"github.com/gorilla/websocket"
	"github.com/ip2location/ip2location-go/v9"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func main() {

	log.Printf("GOMAXPROCS=%d NumCPU=%d", runtime.GOMAXPROCS(0), runtime.NumCPU())

	initCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := godotenv.Load()
	if err != nil {
		logger.Error("failed to load .env file", "error", err)
		os.Exit(1)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	db, err := initDB(&cfg.DB)
	if err != nil {
		logger.Error("failed to initialize db", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	rdb := redis.NewClient(&redis.Options{
		Addr:     "valkey:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	validate := utils.SetupValidator()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if name == "-" {
			return ""
		}

		return name
	})

	e := echo.New()

	// Geo serrvice
	ip2locationDb, err := ip2location.OpenDB("./IP2LOCATION-LITE-DB3.IPV6.BIN")
	if err != nil {
		logger.Error("failed to open geo db", "error", err)
		os.Exit(1)
	}
	defer ip2locationDb.Close()

	geoService := geo.NewGeoService(ip2locationDb)

	// Market related
	transactionManager := market.NewTransactionManager(rdb, db, logger)
	marketStateManager := market.NewStateManager(context.TODO(), rdb, db, logger)
	marketStateManager.Start()
	adminManager := market.NewAdminManager(db)
	betManager := market.NewBetManager(db)
	queryManager := market.NewQueryManager(db, rdb)
	betLiveManager := market.NewBetLiveManager(context.TODO(), rdb, db, logger)
	statsManager := market.NewStatManager(context.TODO(), rdb, db, logger)
	statsManager.Start(context.TODO())

	if err = betLiveManager.PrepopulateLatestBets(initCtx); err != nil {
		logger.Error("failed to prepopulate latest bets", "error", err)
		os.Exit(1)
	}

	if err = betLiveManager.PrepopulateHighBets(initCtx); err != nil {
		logger.Error("failed to prepopulate high bets", "error", err)
		os.Exit(1)
	}

	betLiveManager.Start()

	// Chat
	chatManager := chat.NewChatManager(rdb, db, logger)
	if err = chatManager.PrepopulateChatRooms(initCtx); err != nil {
		logger.Error("failed to prepopulate chat rooms", "error", err)
		os.Exit(1)
	}

	// Initialize sockets
	wsHandlers := handlers.NewWsHandler(betLiveManager, marketStateManager, chatManager, validate)
	hub := ws.NewHub(context.TODO(), rdb)
	wsRouter := ws.NewSocketRouter(validate)
	wsRouter.AddRouteHandler("market:subscribe", wsHandlers.HandleJoinMarketRooms)
	wsRouter.AddRouteHandler("bets:subscribe", wsHandlers.HandleJoinBetsRoom)
	wsRouter.AddRouteHandler("chat:subscribe", wsHandlers.HandleJoinChatRoom)
	wsRouter.AddRouteHandler("chat:send", wsHandlers.RequireAuthentication(wsHandlers.HandleSendMessage))

	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Repos
	userRepo := repos.NewUserRepo(db)
	sessionRepo := repos.NewSessionRepo(db)
	tokenRepo := repos.NewTokenRepo(db)
	commentRepo := repos.NewCommentRepo(db)

	// Middlewares
	authMiddleware := middlewares.NewAuthMiddleware(sessionRepo, validate)

	// HTTP Handlers
	wstHttpHandler := handlers.NewSocketHandler(hub, wsRouter, upgrader)
	authHandler, err := handlers.NewAuthHandler(initCtx, validate, logger, userRepo, sessionRepo, tokenRepo, geoService)
	if err != nil {
		log.Fatal("Error initializing auth handler:", err)
	}

	transactionHandler := handlers.NewTransactionHandler(validate, transactionManager)
	adminMarketHandler := handlers.NewAdminMarketHandler(validate, adminManager, transactionManager)
	adminBetHandler := handlers.NewAdminBetHandler(validate, betManager)
	marketHandler := handlers.NewMarketHandler(validate, marketStateManager, betManager, queryManager, betLiveManager)
	commentHandler := handlers.NewCommentHandler(validate, commentRepo)

	// Register routes
	e.GET("/ws", authMiddleware.Authenticate(wstHttpHandler.ServeWS))

	e.GET("/auth/:provider", authHandler.ProviderLogin)
	e.GET("/auth/:provider/callback", authHandler.GetAuthCallback)
	e.POST("/auth/register", authHandler.RegisterUserByEmail)
	e.POST("/auth/login", authHandler.LoginUserByEmail)

	e.GET("/market/quote", marketHandler.GetQuote)
	e.POST("/market/bet", authMiddleware.RequireAuthentication(transactionHandler.PlaceBet))
	e.GET("/my/bets", authMiddleware.RequireAuthentication(marketHandler.GetBetsUser))
	e.GET("/market/search", authMiddleware.RequireAuthentication(marketHandler.GetMarketsUser))

	e.POST("/comments", authMiddleware.RequireAuthentication(commentHandler.PostComment))
	e.DELETE("/comments", authMiddleware.RequireAuthentication(commentHandler.UserDeleteComment))
	e.GET("/comments", commentHandler.UserGetComments)

	e.POST("/admin/market", authMiddleware.RequireRole(adminMarketHandler.CreateMarket, repos.AdminRole))
	e.DELETE("admin/comments", authMiddleware.RequireRole(commentHandler.AdminDeleteComment, repos.AdminRole))
	e.GET("admin/bets", authMiddleware.RequireRole(adminBetHandler.GetBetsAdmin, repos.AdminRole))

	e.Start(":4000")

}

func initDB(cfg *config.DBConfig) (*pgxpool.Pool, error) {

	ctx := context.TODO()
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to parse runtime DSN: %w", err)
	}

	// Apply runtime pool settings from config
	poolCfg.MaxConns = int32(cfg.MaxConns)
	poolCfg.MinConns = int32(cfg.MinConns)
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime pool: %w", err)
	}

	if err = db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping runtime db: %w", err)
	}

	return db, nil
}
