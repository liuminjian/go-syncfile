package handler

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"go-syncfile/config"
	pb "go-syncfile/grpc"
	"go-syncfile/utils"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// 监控文件管理器
type WatchMgr struct {
	watcher   *fsnotify.Watcher
	watchList sync.Map
}

// 监控文件管理器构造方法
func NewWatchMgr() (*WatchMgr, error) {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &WatchMgr{
		watcher: watch,
	}, nil
}

// 客户端对象
type WatchClient struct {
	eventChan chan fsnotify.Event
}

// 客户端对象构造方法
func NewWatchClient() *WatchClient {
	return &WatchClient{eventChan: make(chan fsnotify.Event)}
}

// 监控客户端请求的目录，返回事件
func (w *WatchMgr) WatchFile(file *pb.File, stream pb.WatchMgr_WatchFileServer) (err error) {
	log.Infof("client watch path:%s, recursive:%t", file.Name, file.Recursive)
	// 判断路径是否存在
	if ok, _ := PathExists(file.Name); !ok {
		log.Errorf("[%s] not exist", file.Name)
		return fmt.Errorf("[%s] not exist", file.Name)
	}
	// 创建客户端对象
	client := NewWatchClient()
	// 判断是否需要监控子目录
	if file.Recursive {
		err = w.AddWatchPathPrefix(file.Name, client)
	} else {
		err = w.AddWatchPath(file.Name, client)
	}
	if err != nil {
		return err
	}

	// 打印所有监控目录
	//w.PrintWatchList()

	// 创建事件channel和结束channel
	eventChan := make(chan fsnotify.Event)
	done := make(chan bool)

	// 因为写操作事件很频繁，所以要合并事件减少返回事件
	go w.mergeFileEvent(client, eventChan, done)

	for {
		select {
		// 当grpc客户端结束时，清理监控目录的客户端
		case <-stream.Context().Done():
			{
				log.Info("client exit.")
				// 判断是否需要清理监控子目录
				if file.Recursive {
					err = w.RemoveWatchClientPrefix(file.Name, client)
				} else {
					err = w.RemoveWatchClient(file.Name, client)
				}
				done <- true
				w.PrintWatchList()
				return
			}
		// 接收合并完的事件后发送给grpc客户端
		case ev := <-eventChan:
			{
				log.Infof("get event:%s", file.Name)
				if err = stream.Send(&pb.FileEvent{Message: ev.Name}); err != nil {
					log.Error(err)
					done <- true
					return
				}
			}
		}

	}

}

// 监控客户端请求的目录，并且将文件用rsync同步给客户端
func (w *WatchMgr) WatchFileAndSync(file *pb.SyncFile, stream pb.WatchMgr_WatchFileAndSyncServer) (err error) {
	log.Infof("client watch path:%s, recursive:%t", file.Name, file.Recursive)
	// 判断路径是否存在
	if ok, _ := PathExists(file.Name); !ok {
		log.Errorf("[%s] not exist", file.Name)
		return fmt.Errorf("[%s] not exist", file.Name)
	}
	// 创建客户端对象
	client := NewWatchClient()
	// 判断是否需要监控子目录
	if file.Recursive {
		err = w.AddWatchPathPrefix(file.Name, client)
	} else {
		err = w.AddWatchPath(file.Name, client)
	}
	if err != nil {
		return err
	}

	// 打印所有监控目录
	//w.PrintWatchList()

	// 创建事件channel和结束channel
	eventChan := make(chan fsnotify.Event)
	done := make(chan bool)
	// 创建cli对象
	cli := utils.NewCli(file.Ip, file.Port, file.User, file.Password)

	// 因为写操作事件很频繁，所以要合并事件减少返回事件
	go w.mergeFileEvent(client, eventChan, done)

	for {
		select {
		// 当grpc客户端结束时，清理监控目录的客户端
		case <-stream.Context().Done():
			{
				log.Info("client exit.")
				// 判断是否需要清理监控子目录
				if file.Recursive {
					err = w.RemoveWatchClientPrefix(file.Name, client)
				} else {
					err = w.RemoveWatchClient(file.Name, client)
				}
				done <- true
				w.PrintWatchList()
				return
			}
		// 接收合并完的事件后发送给grpc客户端
		case ev := <-eventChan:
			{
				// 文件被删就不同步
				if ok, _ := PathExists(ev.Name); !ok {
					continue
				}
				log.Debugf("get event:%s", ev.Name)

				// 获取远程目录
				remotePath := strings.TrimPrefix(ev.Name, file.Name)
				remotePath = path.Join(file.RemotePath, remotePath)
				// 执行rsync命令
				cli.CreateRsyncCommandWithSShPass(ev.Name, remotePath)
				err = cli.Exec()

				if err == nil {
					err = stream.Send(&pb.SyncFileResp{Code: 0, Output: cli.Output, Name: ev.Name})
				} else {
					log.Error(err)
					err = stream.Send(&pb.SyncFileResp{Code: 1, Output: cli.Output, Error: cli.Error, Name: ev.Name})
				}

				if err != nil {
					log.Error(err)
					done <- true
					return
				}
			}
		}

	}
}

// 检查目录是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// 合并文件事件，在短时间内如果频繁对同一个文件操作合并为一个事件
func (w *WatchMgr) mergeFileEvent(client *WatchClient, eventChan chan fsnotify.Event, done chan bool) {
	// 创建一个计时器，延迟事件发送，假如短时间内频繁事件则计时器重新计时
	var lastModifiedEvent fsnotify.Event
	var timer *time.Timer
	count := 0
	maxCount := config.ConfigMgr.GetIntDefault("server.mergeEventCount", 30)
	timeout := config.ConfigMgr.GetDurationDefault("server.mergeEventTimeout", 1000*time.Microsecond)
	for {
		select {
		case ev := <-client.eventChan:
			{

				if timer != nil && lastModifiedEvent.Name != ev.Name {
					// 不同文件的事件则马上发送
					if timer.Stop() {
						count = 0
						eventChan <- lastModifiedEvent
					}

					timer = time.AfterFunc(timeout, func(event fsnotify.Event) func() {
						return func() {
							count = 0
							eventChan <- event
						}
					}(ev))
					count = 1
				} else if timer != nil && lastModifiedEvent.Name == ev.Name {
					// 相同文件事件则计数，一定数量后发送
					count += 1
					if count >= maxCount {
						count = 0
						timer.Stop()
						timer = nil
						eventChan <- ev
					} else {
						if !timer.Reset(timeout) {
							count = 1
						}
					}

				} else if timer == nil {
					// 第一次开始时创建计时器
					timer = time.AfterFunc(timeout, func(event fsnotify.Event) func() {
						return func() {
							count = 0
							eventChan <- event
						}
					}(ev))
					count = 1
				}
				lastModifiedEvent = ev
			}
		case <-done:
			{
				//log.Info("safe exit.")
				return
			}

		}
	}
}

// 添加监控的目录
func (w *WatchMgr) AddWatchPath(path string, client *WatchClient) error {
	if clients, ok := w.watchList.Load(path); ok {
		w.watchList.Store(path, append(clients.([]*WatchClient), client))
	} else {
		w.watchList.Store(path, []*WatchClient{client})
	}

	err := w.watcher.Add(path)

	if err != nil {
		return err
	}

	return nil
}

// 添加监控的目录，并且递归这个目录下一层
func (w *WatchMgr) AddWatchPathPrefix(path string, client *WatchClient) (err error) {
	err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			pathName, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			err = w.watcher.Add(pathName)
			if err != nil {
				return err
			}

			if clients, ok := w.watchList.Load(pathName); ok {
				w.watchList.Store(pathName, append(clients.([]*WatchClient), client))
			} else {
				w.watchList.Store(pathName, []*WatchClient{client})
			}
		}
		return nil
	})
	return
}

// 删除监控目录的客户端
func (w *WatchMgr) RemoveWatchClient(path string, client *WatchClient) error {

	if clients, ok := w.watchList.Load(path); ok {
		clientList := clients.([]*WatchClient)
		for index, c := range clientList {
			if c == client {
				tmp := append(clientList[:index], clientList[index+1:]...)
				w.watchList.Store(path, tmp)
				if len(tmp) == 0 {
					err := w.watcher.Remove(path)
					w.watchList.Delete(path)
					return err
				}
				return nil
			}
		}

	}

	return nil
}

// 删除监控对应目录或以目录为前缀的客户端
func (w *WatchMgr) RemoveWatchClientPrefix(path string, client *WatchClient) (err error) {

	w.watchList.Range(func(key, value interface{}) bool {
		clients := value.([]*WatchClient)
		pathName := key.(string)
		if strings.HasPrefix(pathName, path) {
			for index, c := range clients {
				if c == client {
					tmp := append(clients[:index], clients[index+1:]...)
					w.watchList.Store(pathName, tmp)
					if len(tmp) == 0 {
						err = w.watcher.Remove(pathName)
						w.watchList.Delete(pathName)
						log.Debugf("remove %s", pathName)
					}
				}
			}
		}
		return true
	})

	return
}

// 打印调试监控的目录
func (w *WatchMgr) PrintWatchList() {
	w.watchList.Range(func(key, value interface{}) bool {
		log.Debugf("watch: %s", key)
		return true
	})
}

// 开始监控，将文件事件发给要监控的客户端
func (w *WatchMgr) StartWatch() {
	go func() {
		for {
			select {
			case ev := <-w.watcher.Events:
				{
					if ev.Op&fsnotify.Create == fsnotify.Create {
						log.Debug("创建文件：", ev.Name)
					}
					if ev.Op&fsnotify.Write == fsnotify.Write {
						log.Debug("写文件：", ev.Name)
					}
					if ev.Op&fsnotify.Remove == fsnotify.Remove {
						log.Debug("删文件：", ev.Name)
					}
					if ev.Op&fsnotify.Rename == fsnotify.Rename {
						log.Debug("重命名文件：", ev.Name)
					}
					if ev.Op&fsnotify.Chmod == fsnotify.Chmod {
						log.Debug("修改权限：", ev.Name)
					}

					clients, ok := w.watchList.Load(ev.Name)
					if ok {
						for _, c := range clients.([]*WatchClient) {
							c.eventChan <- ev
						}
					} else {

						w.watchList.Range(func(key, value interface{}) bool {
							clients := value.([]*WatchClient)
							pathName := key.(string)
							if strings.HasPrefix(ev.Name, pathName) {
								for _, c := range clients {
									c.eventChan <- ev
								}
							}
							return true
						})

					}

				}
			case err := <-w.watcher.Errors:

				{
					log.Println("error:", err)
				}
			}
		}
	}()
}
