package utils

import "fmt"

func ExampleCli_CreateRsyncCommandWithSShPass() {
	cli := NewCli("192.168.1.224", 22, "root", "xxx")
	cli.CreateRsyncCommandWithSShPass("./cli.go",
		"/home/duni/rsyncpath/cli.go")
	err := cli.Exec()
	fmt.Println(err)
	// Output:
	// <nil>
}
