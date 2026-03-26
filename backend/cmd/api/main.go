package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Mono303/Huzhoumahjong/backend/internal/config"
	"github.com/Mono303/Huzhoumahjong/backend/internal/handler"
	"github.com/Mono303/Huzhoumahjong/backend/internal/repository/postgres"
	redisrepo "github.com/Mono303/Huzhoumahjong/backend/internal/repository/redis"
	"github.com/Mono303/Huzhoumahjong/backend/internal/service"
	wsserver "github.com/Mono303/Huzhoumahjong/backend/internal/ws"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	cache := redisrepo.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err := cache.Ping(ctx); err != nil {
		log.Fatalf("ping redis: %v", err)
	}

	userRepo := postgres.NewUserRepository(db)
	roomRepo := postgres.NewRoomRepository(db)
	matchRepo := postgres.NewMatchRepository(db)
	hub := wsserver.NewHub()

	userService := service.NewUserService(userRepo, cache, cfg.SessionTTL)
	roomService := service.NewRoomService(cfg, roomRepo, matchRepo, cache, hub)

	router := gin.Default()
	router.Use(cors())

	api := handler.New(userService, roomService, hub)
	api.Register(router)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("backend listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
