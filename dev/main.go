package main

import (
	"fmt"
	"os"

	"../../DNS"
)

// const cfgPath = "../config.json"

func main() {
	if len(os.Args) != 2 {
		fmt.Println("usage: server [path to config file]")
		return
	}

	cfgPath := os.Args[1]
	cfg := DNS.ReadConfig(cfgPath)
	server := DNS.NewServer(*cfg)
	server.Run()
}
