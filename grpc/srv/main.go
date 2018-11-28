package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libs/grpc/grpclb/consul"
	pb "github.com/libs/grpc/hello"
	"github.com/libs/grpc/interceptor"
	"github.com/libs/grpc/interceptor/logging"
	"github.com/libs/grpc/interceptor/ratelimit"
	"github.com/libs/grpc/interceptor/recovery"
	"github.com/libs/grpc/interceptor/tracing"
	"github.com/libs/grpc/interceptor/tracing/opentracing"
	"github.com/libs/grpc/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

var (
	endpoint = "127.0.0.1:8500"
)

type server struct {
}

func (s *server) Hello(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	str := fmt.Sprintf("Hello %s !", req.Name)
	//tracing
	var resp struct {
		HorizonVersion      string `json:"horizon_version"`
		CoreVersion         string `json:"core_version"`
		HistoryLatestLedger int64  `json:"history_latest_ledger"`
		HistoryElderLedger  int64  `json:"history_elder_ledger"`
		CoreLatestLedger    int64  `json:"core_latest_ledger"`
		NetworkPassphrase   string `json:"network_passphrase"`
		ProtocolVersion     int64  `json:"protocol_version"`
	}
	if err := util.HttpCall("GET", "https://horizon-testnet.stellar.org", "", &resp, ctx); err != nil {
		fmt.Printf("httpcall err: %v\n", err)
	}
	d, _ := json.Marshal(resp)
	fmt.Println("resp1:", string(d))

	time.Sleep(1 * time.Second)
	return &pb.Response{Msg: str}, nil
}

func main() {
	var (
		port     int
		host     = "127.0.0.1" //todo
		servName = "localhost" //hello_service
	)
	flag.IntVar(&port, "port", 5001, "server listening port")
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := consul.Register(servName, host, port, endpoint, 3*time.Second, 5); err != nil {
		log.Fatalf("Register server err: %v", err)
	}
	creds, err := credentials.NewServerTLSFromFile("../tls/ca.pem", "../tls/server.key")
	if err != nil {
		log.Fatalf("NewServerTLSFromFile err: %v", err)
	}
	tracer, closer := tracing.Init("grpc-srv")
	defer closer.Close()
	s := grpc.NewServer(grpc.Creds(creds),
		interceptor.WithUnaryServerChain(
			ratelimit.UnaryServerInterceptor(),
			opentracing.UnaryServerInterceptor(opentracing.WithTracer(tracer)),
			//initial.UnaryServerInterceptor(),
			recovery.UnaryServerInterceptor(),
			logging.UnaryServerInterceptor(logging.DefaultZapLogger()),
		))
	pb.RegisterSayServer(s, &server{})

	go gracefulStop(s)

	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func gracefulStop(s *grpc.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGQUIT)
	<-quit

	log.Println("Shutdown server...")
	consul.Unregister()
	s.GracefulStop()
	log.Println("Server shutdown successfully")
}
