# go-syncfile
inotify+grpc+rsync做的文件远程同步工具

## Quick Start

#### Start server
```go
      listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
      grpcServer := grpc.NewServer()
      mgr, err := handler.NewWatchMgr()
      mgr.StartWatch()
      pb.RegisterWatchMgrServer(grpcServer, mgr)
      grpcServer.Serve(listener)
```
#### Start client
```go
      conn, err := grpc.Dial(fmt.Sprintf("localhost:%d", port), grpc.WithInsecure())
      defer conn.Close()
      client := pb.NewWatchMgrClient(conn)
      localPath, _ = filepath.Abs(".")
      file := &pb.SyncFile{
        Name:       localPath,
        Recursive:  true,
        Ip:         "xxxxx",
        Port:       22,
        User:       "xxxx",
        Password:   "xxxx",
        RemotePath: "xxxx",
      }
      stream, err := client.WatchFileAndSync(context.Background(), file)
```
