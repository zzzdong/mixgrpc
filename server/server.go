// protoc -I=./proto --go_out=plugins=grpc:proto proto/hello.proto
//

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	pb "mixgrpc/proto"

	"github.com/go-chi/chi"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type MyServer struct {
}

func (s *MyServer) Greeting(ctx context.Context, req *pb.GreetingReq) (*pb.GreetingRsp, error) {
	var rsp pb.GreetingRsp

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		log.Println("metadata=>", md)
	}

	rsp.Message = fmt.Sprintf("Hello, %s!", req.Name)

	header := metadata.Pairs("errcode", "0")
	tailer := metadata.Pairs("errcode", "0", "errmsg", "ok")

	grpc.SendHeader(ctx, header)
	grpc.SetTrailer(ctx, tailer)

	return &rsp, nil
}

type ServerMux struct {
	h1Router  http.Handler
	h2Handler http.Handler
}

func NewServerMux(grpcServer *grpc.Server) *ServerMux {
	// HTTP1 handler
	h1Router := chi.NewRouter()

	// HTTP2 handler
	h2s := &http2.Server{}
	h2Handler := h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
			return
		}
		log.Println("not grpc on http2")
	}), h2s)

	h1Router.Get("/_/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	return &ServerMux{
		h1Router:  h1Router,
		h2Handler: h2Handler,
	}
}

func (s *ServerMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("recover", r)
		}
	}()

	if r.ProtoMajor == 2 {
		s.h2Handler.ServeHTTP(w, r)
	} else {
		s.h1Router.ServeHTTP(w, r)
	}
}

func mixServer() {
	grpcServer := grpc.NewServer()
	pb.RegisterHelloServer(grpcServer, &MyServer{})

	srv := NewServerMux(grpcServer)

	err := http.ListenAndServe("localhost:8099", srv)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
}

func grpcServer() {
	grpcServer := grpc.NewServer()
	pb.RegisterHelloServer(grpcServer, &MyServer{})

	listener, err := net.Listen("tcp", "localhost:8099")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		return
	}

	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalln("grpc.Server failed", err)
		return
	}
}

func main() {
	// grpcServer()
	mixServer()
}
