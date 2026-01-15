package app

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"router-manager/internal/config"
	"router-manager/internal/pb"
	"router-manager/internal/repository/postgres"
	"router-manager/internal/repository/redis"
	"router-manager/internal/service"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Application struct {
	service *service.CommandService

	pg  *config.Postgres
	red *config.Redis

	grpcServer *grpc.Server
	httpServer *http.Server
}

func NewApplication() *Application {
	app := &Application{}

	app.pg = config.NewPostgres()
	log.Println("Connected to PostgreSQL")

	app.red = config.InitRedis()
	log.Println("Connected to Redis")

	pgRepo := postgres.NewPostgresRepository(app.pg.Pool)
	redRepo := redis.NewRedisRepository(app.red.Client)

	app.service = service.NewCommandService(pgRepo, redRepo)

	app.grpcServer = grpc.NewServer()
	pb.RegisterCommandServiceServer(app.grpcServer, app.service)

	mux := runtime.NewServeMux()

	mux.HandlePath("GET", "/metrics", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		promhttp.Handler().ServeHTTP(w, r)
	})

	ctx := context.Background()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	err := pb.RegisterCommandServiceHandlerFromEndpoint(ctx, mux, "localhost:50051", opts)
	if err != nil {
		log.Fatalf("Couldn't register REST handler: %v", err)
	}

	app.httpServer = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	return app
}

func (a *Application) Run() {
	go func() {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("Failed to listen gRPC: %v", err)
		}
		log.Println("gRPC server running on :50051")
		if err := a.grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()

	go func() {
		log.Println("REST API running on :8080")
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down...")

	a.grpcServer.GracefulStop()
	if err := a.httpServer.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}

	if a.pg != nil {
		a.pg.Close()
		log.Println("PostgreSQL connection closed")
	}
	if a.red != nil {
		a.red.Close()
		log.Println("Redis connection closed")
	}

	log.Println("Application stopped")
}
