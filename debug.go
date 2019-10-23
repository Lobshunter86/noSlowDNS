package DNS

import "fmt"

const DEBUG = 0

func Debug(a ...interface{}) {
	if DEBUG != 0 {
		fmt.Println(a...)
	}
}
