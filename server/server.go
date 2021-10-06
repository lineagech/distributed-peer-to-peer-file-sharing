package main

import (
    "fmt"
    "connect"
)

func main() {
    message := connect.TestMsg();
    fmt.Println("Server prog...", message);

    connect.RunServer("localhost", "9527")
}
