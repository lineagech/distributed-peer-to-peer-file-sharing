package main

import (
    "fmt"
    "connect"
)

var ServerAddr = []string{"127.0.0.1", "9527"}

func main() {
    message := connect.TestMsg();
    fmt.Println("Server prog...", message);

    connect.RunServer(ServerAddr[0], ServerAddr[1])
}
