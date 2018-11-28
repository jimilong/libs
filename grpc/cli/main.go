package main

import (
	"context"
	"log"
	"time"

	"github.com/libs/grpc/grpclb/consul"
	"github.com/libs/grpc/interceptor"
	"github.com/libs/grpc/interceptor/logging"
	"github.com/libs/grpc/interceptor/tracing"
	"github.com/libs/grpc/interceptor/tracing/opentracing"

	pb "github.com/libs/grpc/hello"

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/resolver"
)

var (
	endpoint = "127.0.0.1:8500"
	servName = "localhost" //hello_service
)

func main() {
	r := consul.NewResolver(endpoint)
	resolver.Register(r)

	creds, err := credentials.NewClientTLSFromFile("../tls/ca.pem", "")
	if err != nil {
		log.Fatalf("NewServerTLSFromFile err: %v", err)
	}
	tracer, closer := tracing.Init("grpc-cli")
	defer closer.Close()
	conn, err := grpc.Dial(
		r.Scheme()+":///"+servName,
		//grpc.WithInsecure(),
		grpc.WithTransportCredentials(creds),
		//grpc.WithBlock(),
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithUnaryInterceptor(interceptor.ChainUnaryClient(
			opentracing.UnaryClientInterceptor(opentracing.WithTracer(tracer)),
			//initial.UnaryClientInterceptor(),
			logging.UnaryClientInterceptor(logging.DefaultZapLogger()),
		)),
	)
	if err != nil {
		log.Fatalf("did not connect target: %v\n", err)
	}
	defer conn.Close()

	cli := pb.NewSayClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := cli.Hello(ctx, &pb.Request{Name: "longmin1"})
	if err != nil {
		log.Fatalf("could not say %v\n", err)
	}
	log.Println("say resp:", resp.Msg)
}
