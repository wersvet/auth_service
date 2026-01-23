package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"auth-service/internal/db"
	grpcserver "auth-service/internal/grpc"
	"auth-service/internal/handlers"
	"auth-service/internal/metrics"
	"auth-service/internal/rabbitmq"
	"auth-service/internal/telemetry"
	authpb "auth-service/proto/auth"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	amqpURL := envWithDefault("AMQP_URL", "amqp://guest:guest@localhost:5672/")
	logsExchange := envWithDefault("LOGS_EXCHANGE", "logs.events")
	serviceName := envWithDefault("SERVICE_NAME", "auth-service")
	environment := envWithDefault("ENVIRONMENT", "local")

	publisher := rabbitmq.NewPublisher(amqpURL, logsExchange)
	auditEmitter := telemetry.NewAuditEmitter(publisher, serviceName, environment)
	metricsCollector := metrics.New(serviceName)

	handler := handlers.NewAuthHandler(database, jwtSecret, auditEmitter, metricsCollector)

	router := gin.Default()
	router.Use(gin.Logger(), gin.Recovery(), metricsCollector.Middleware())

	router.GET("/metrics", gin.WrapH(metricsCollector.Handler()))
	router.POST("/auth/register", handler.Register)
	router.POST("/auth/login", handler.Login)
	router.GET("/auth/validate", handler.ValidateToken)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	grpcAddr := ":8084"
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", grpcAddr, err)
	}

	grpcSrv := grpc.NewServer()
	authpb.RegisterAuthServiceServer(grpcSrv, grpcserver.NewAuthGRPCServer(database, jwtSecret))

	go func() {
		log.Printf("starting auth-service gRPC on %s", grpcAddr)
		if err := grpcSrv.Serve(grpcListener); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	go func() {
		log.Printf("starting auth-service HTTP on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	grpcSrv.GracefulStop()
	publisher.Close()
	log.Println("servers stopped gracefully")
}

func envWithDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
