package main

import (
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"log"
	"net"
	"net/http"
)

func main() {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	mux := http.NewServeMux()
	hub := internal.NewHub()

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	lis, err := net.Listen("tcp", "0.0.0.0:8081")
	if err != nil {
		log.Fatal(err)
		return
	}

	go func() {
		log.Printf("listening on 8081")
		err := grpcServer.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	internal.RegisterCourierServer(grpcServer, &internal.CourierServerImpl{Hub: hub})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade error: %v\n", err)
			return
		}

		ws := hub.RegisterConnection(conn)
		log.Printf("registering new websocket %s\n", ws.ID)
	})

	log.Printf("listening on 8080")
	if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
		panic(err)
	}

	grpcServer.GracefulStop()
}
