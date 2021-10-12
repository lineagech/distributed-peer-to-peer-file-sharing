package main

import (
    "fmt"
    "connect"
    "snapshot"
    "messages"
    "flag"
)

var ServerAddr = []string{"127.0.0.1", "9527"}
var resumePtr *bool

func WriteSnapshotCallBack(filename string, length int) {
    snapshot.RegisterFileP2PServer(filename, length)
}

func WriteFileLocationsSnapshotCallBack(filename string, chunk_id int, locations []string) {
    snapshot.WriteChunkLocationsSnapshot(filename, chunk_id, locations)
}

func resumeProcess() {
    file_list, len_list := snapshot.ReadP2PSnapshot()
    messages.SetP2PServerFileList(file_list, len_list)
    for _, filename := range file_list {
        loc := snapshot.ReadFileLocP2PSnapshot(filename)
        messages.SetFileLoc(filename, loc)
    }
    fmt.Println("\n----------------------- P2P Server Resume Process Done ----------------------\n")
}

func cmdArgs() {
    resumePtr = flag.Bool("resume", false, "Resume")
    flag.Parse()
}

func main() {
    fmt.Println("Start P2P Server...");
    cmdArgs()
    if *resumePtr == true {
        resumeProcess()
    }

    messages.RegisterSnapshotCallBack(WriteSnapshotCallBack)
    messages.RegisterFileLocSnapshotCallBack(WriteFileLocationsSnapshotCallBack)
    connect.RunServer(ServerAddr[0], ServerAddr[1])
}
