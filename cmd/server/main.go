package main

import (
	"log/slog"
	"net"
	"os"
	"os/signal"

	"github.com/shadiestgoat/bankDataDB/config"
	"github.com/shadiestgoat/bankDataDB/grpc/bank_data"
	"github.com/shadiestgoat/bankDataDB/grpc/user_svc"
	"github.com/shadiestgoat/bankDataDB/pb/bank_svc_pb"
	"github.com/shadiestgoat/bankDataDB/pb/user_svc_pb"
	"google.golang.org/grpc"

	_ "github.com/shadiestgoat/bankDataDB/bank_parser/all"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

func main() {
	cleanup := config.LoadBasics()
	defer func () {
		err := cleanup()
		if err != nil {
			slog.Error("Cleanup error", "error", err)
		}
	}()

	grpcSRV := grpc.NewServer(
		grpc.UnaryInterceptor(bank_data.NewAuthInterceptor()),
	)
	bank_svc_pb.RegisterBankDataServer(grpcSRV, bank_data.NewAPI())
	user_svc_pb.RegisterUserServiceServer(grpcSRV, user_svc.NewAPI())

	var lis net.Listener
	var err error

	if path, ok := os.LookupEnv("GRPC_UNIX_PATH"); ok {
		lis, err = net.Listen("unix", path)
	} else {
		port := os.Getenv("PORT")
		if port == "" {
			port = "3000"
		}
		lis, err = net.Listen("tcp", port)
	}

	if err != nil {
		panic(err)
	}
	

	grpcEnded := make(chan bool)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		err = grpcSRV.Serve(lis)
		if err != nil {
			slog.Error("GRPC Ended with error", "error", err)
			close(grpcEnded)
		}
	}()

	select {
	case <-c:
		grpcSRV.GracefulStop()
	case <-grpcEnded:
		panic("Existing server (not happy)...")
	}
}
