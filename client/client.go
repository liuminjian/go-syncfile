package main

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	_ "go-syncfile/config"
	pb "go-syncfile/grpc"
	"google.golang.org/grpc"
	"io"
	"path/filepath"
)

func main() {
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", 8080), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	log.Info("start rpc client...")

	client := pb.NewWatchMgrClient(conn)

	localPath := "."
	localPath, _ = filepath.Abs(localPath)

	//file := &pb.File{Name: localPath, Recursive:true}
	file := &pb.SyncFile{
		Name:       localPath,
		Recursive:  true,
		Ip:         "192.168.1.224",
		Port:       22,
		User:       "root",
		Password:   "xxxx",
		RemotePath: "/home/duni/rsyncpath",
	}

	//stream, err := client.WatchFile(context.Background(), file)
	stream, err := client.WatchFileAndSync(context.Background(), file)

	if err != nil {
		log.Fatal(err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if resp.Code == 0 {
			log.Infof("sync file [%s] success.", resp.Name)
		} else {
			log.Infof("sync file [%s] fail. err:%s", resp.Name, resp.Error)
		}
	}
}
