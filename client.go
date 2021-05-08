package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	pb "mixgrpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func main() {
	conn, err := grpc.Dial("localhost:8099", grpc.WithInsecure())
	if err != nil {
		log.Fatal("client error", err)
		return
	}

	client := pb.NewHelloClient(conn)

	gCtx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	req := pb.GreetingReq{
		Name: "Tom",
	}

	wg := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			md := metadata.New(map[string]string{"key1": "hello", "key2": "world", "brick-caller": "account", "brick-call-depth": "1", "brick-trace-id": "xxxxx"})
			ctx := metadata.NewOutgoingContext(gCtx, md)

			var header = metadata.Pairs()
			var tailer = metadata.Pairs()

			resp, err := client.Greeting(ctx, &req, grpc.Header(&header), grpc.Trailer(&tailer))
			if err != nil {
				log.Fatal("greeting failed", err)
				return
			}

			log.Printf("greeting: %s, header: %+v, tailer: %+v \n", resp.Message, header, tailer)

			wg.Done()
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {

			resp, err := http.DefaultClient.Get("http://localhost:8099/_/ping")
			if err != nil {
				log.Println("http request failed, ", err)
				return
			}

			log.Println("http response, StatusCode:", resp.StatusCode)

			wg.Done()

		}()
	}

	wg.Wait()
}
