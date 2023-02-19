package main

import (
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"time"
)

var (
	// The hostname that the GRPC server will be served from. This will be used to create the
	// client's webhook URL and does not signify what address the server will use on this machine
	hostname string

	// The address that the GRPC server will be listening on
	grpcListenAddress string

	// The address that the websocket server will be listening on
	websocketListenAddress string

	jwtSecret []byte
)

func init() {
	// Set up logrus
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	})

	hostname = internal.MustGetEnv("HOSTNAME")
	grpcListenAddress = internal.MustGetEnv("GRPC_LISTEN_ADDRESS")
	websocketListenAddress = internal.MustGetEnv("WEBSOCKET_LISTEN_ADDRESS")
	jwtSecret = []byte(internal.MustGetEnv("JWT_SECRET"))
}

func main() {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	mux := http.NewServeMux()
	hub := internal.NewHub(hostname)

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	lis, err := net.Listen("tcp", grpcListenAddress)
	if err != nil {
		panic(err)
		return
	}

	go func() {
		log.Infoln("listening on 8081")
		err := grpcServer.Serve(lis)
		if err != nil {
			panic(err)
		}
	}()

	internal.RegisterCourierServer(grpcServer, &internal.CourierServerImpl{Hub: hub})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := internal.GetAndVerifyJWT(jwtSecret, r); err != nil {
			w.WriteHeader(403)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Warnln("upgrade error")
			return
		}

		ws := hub.RegisterConnection(conn)
		log.WithFields(log.Fields{"id": ws.ID}).Infoln("registering new websocket")
	})

	log.Infoln("listening on 8080")
	if err := http.ListenAndServe(websocketListenAddress, mux); err != nil {
		panic(err)
	}

	grpcServer.GracefulStop()
}
