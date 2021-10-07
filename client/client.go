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

type PeerRequest int

const (
    RegisterRequest PeerRequest = iota
    FilelistRequest
    FileLocationsRequest
    ChunkRegisterRequest
    FileChunkRequest
)

var fileSharePtr *string
var dirPtr *string

func cmdArgs() {
    fileSharePtr = flag.String("file-to-share", "", "file to share")
    dirPtr = flag.String("dir-to-share", "", "directory to share")
    flag.Parse()
}

func getFilesToShare() string {
    var ret = ""
    if *fileSharePtr != "" {
        ret += *fileSharePtr
    }
    if *dirPtr != "" {
        ret += *dirPtr
    }
    return ret
}

func processPeerRequest() {
    var line string
    fmt.Scanln(&line)

}

func main() {
    cmdArgs()

    conn := connect.ConnectToServer("localhost", "9527")

    connect.SendRegisterRequest(conn, []string{"HelloWorld"}, []int{1024})
    _ = connect.RecvMsg(conn, 0)

}

