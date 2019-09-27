package utils

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
)

type Cli struct {
	Ip       string
	Port     int32
	User     string
	Password string
	Command  string
	Output   string
	Error    string
}

func NewCli(ip string, port int32, user string, password string) *Cli {
	return &Cli{
		Ip:       ip,
		Port:     port,
		User:     user,
		Password: password,
	}
}

func (c *Cli) CreateRsyncCommandWithSShPass(localPath string, remotePath string) {
	cmd := fmt.Sprintf("sshpass -p %s -P %d rsync -az --rsync-path=\"mkdir -p %s && rsync\" %s %s@%s:%s ",
		c.Password, c.Port, filepath.Dir(remotePath), localPath, c.User, c.Ip, remotePath)
	c.Command = cmd
}

func (c *Cli) Exec() error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", c.Command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.Output = stdout.String()
	c.Error = stderr.String()
	return cmd.Run()
}
