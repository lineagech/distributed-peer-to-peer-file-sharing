package main

import (
    _ "net"
    _ "log"
    "fmt"
    _ "bufio"
    _ "os"
    "flag"
    "connect"
    _ "encoding/json"
)

func main() {
    //args := os.Args[1:]
    fileSharePtr := flag.String("file-to-share", "", "file to share")
    dirPtr := flag.String("dir-to-share", "", "directory to share")
    //fileDownloadPtr := flag.String()

    flag.Parse()
    if *fileSharePtr != "" {
        fmt.Println("file: ", *fileSharePtr)
    }
    if *dirPtr != "" {
        fmt.Println("dir: ", *dirPtr)
    }

    conn := connect.ConnectToServer("localhost", "9527")
    //msg := "I am hungry\n"
    //connect.SendMsg(conn, []byte(msg))
    connect.SendRegisterRequest(conn, []string{"HelloWorld"}, []int{1024})
    reponse := connect.RecvMsg(conn, 0)
    fmt.Println(string(reponse))
}

