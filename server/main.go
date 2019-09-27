package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"go-syncfile/config"
	_ "go-syncfile/config"
	pb "go-syncfile/grpc"
	"go-syncfile/handler"
	"google.golang.org/grpc"
	"net"
)

func main() {
	port := config.ConfigMgr.GetIntDefault("server.port", 8080)

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()

	mgr, err := handler.NewWatchMgr()

	if err != nil {
		log.Fatal(err)
	}

	mgr.StartWatch()

	pb.RegisterWatchMgrServer(grpcServer, mgr)

	log.Info("start rpc server...")

	err = grpcServer.Serve(listener)
	if err != nil {
		log.Fatal(err)
	}
}
