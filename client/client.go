package main

import (
    _ "net"
    "fmt"
    "bufio"
    "os"
    "flag"
    "connect"
    "messages"
    "strings"
    "strconv"
    "sort"
    "math/rand"
    "log"
    "sync"
)

type UserInput int
var PeerName string
var ServerAddr  = []string {"127.0.0.1", "9527"}
var PeerAddr []string

const (
    ShareFile UserInput = iota
    DownloadFile
    GetFileList
    Invalid
)

type file_loc_info_t struct {
    Chunk_id int
    Num_locations int
}


type download_file_t struct {
    Chunks map[int][]byte
}

var downloadedFiles = make(map[string]download_file_t)

var ipAddr *string
var fileDownloadPtr *string
var fileSharePtr *string
var dirPtr *string

func dummyFunc(filename string, chunk_id int, locations []string) {
    log.Printf("%s: %s, %d\n", PeerName, filename, chunk_id)
    log.Println(locations)
}

func dummyFunc2(bytes []byte) {
    log.Println("Dummy Recv Bytes ", len(bytes))
}

func cmdArgs() {
    ipAddr = flag.String("ip", "", "ip exposed to the network")
    fileDownloadPtr = flag.String("download", "", "file to download")
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

func parseUserInput(line string) (UserInput, string) {
    var res UserInput = Invalid
    var filename string
    if strings.HasPrefix(line, "Download") {
        res = DownloadFile
    } else if strings.HasPrefix(line, "Share") {
        res = ShareFile
    } else if strings.Compare(line, "Get-file-list") == 0{
        res = GetFileList
        return res, ""
    }
    log.Println("Line ", line)
    split_line := strings.Split(line, " ")
    filename = split_line[1]

    return res, filename
}

func assembleChunks(filename string, num_chunks int) {
    file, err := os.Create(filename)
    //check(err)
    if err != nil {
        log.Println("Assemble Chunks Failed")
    }
    defer file.Close()
    
    log.Println("Assemble Chunks for ", filename)
    for i := 0; i <= num_chunks; i++ {
        file.Write(downloadedFiles[filename].Chunks[i])
    }
    file.Sync()
}

func getFileInfo(filename string) os.FileInfo {
    file_info, err := os.Stat(filename)
    if err != nil {
        fmt.Println("get file info error!")
        return nil
    }
    return file_info
}

func writeToLocalStore(filename string, chunk_id int, bytes_p *[]byte) {
    _, found := downloadedFiles[filename]
    if found != true {
        downloadedFiles[filename] = download_file_t {
            make(map[int][]byte),
        }
    }
    downloadedFiles[filename].Chunks[chunk_id] = *bytes_p

    /* Write to the file */
    chunk_filename := strconv.Itoa(chunk_id) + "_" + filename;
    log.Printf("%s: Write recv bytes to %s\n", PeerName, chunk_filename)
    err := os.WriteFile(chunk_filename, *bytes_p, 0644)
    if err != nil {
        fmt.Println("Chunk File Write Error!")
    }
}

func registerChunk(filename string, chunk_id int, peer_addr string) {
    conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1])
    connect.SendChunkRegisterRequest(conn, filename, chunk_id, PeerName)
    chunk_register_resp := connect.RecvMsg(conn, messages.Chunk_Register_Request).(messages.Chunk_register_response_t)
    if chunk_register_resp.Succ == 1 {
        log.Printf("%s: Register Chunk for %s %d-th chunk Succeeded\n", PeerName, filename, chunk_id)
    } else {
        log.Printf("%s: Register Chunk for %s %d-th chunk Failed\n", PeerName, filename, chunk_id)
    }
}

func recvChunk(wg *sync.WaitGroup, filename string, chunk_id int, locations []string) {
    defer wg.Done()
    dest_peer_addr_str := locations[rand.Int() % len(locations)]
    dest_peer_addr := strings.Split(dest_peer_addr_str, ":")

    //conn := connect.ConnectToServer(dest_peer_addr[0], dest_peer_addr[1], PeerAddr[0], PeerAddr[1])
    conn := connect.ConnectToServer(dest_peer_addr[0], dest_peer_addr[1])
    connect.SendFileChunkRequest(conn, filename, chunk_id)

    log.Printf("%s: Send File Chunk request for %s %d-th Chunk to %s\n", PeerName, filename, chunk_id, dest_peer_addr_str)

    recv_chunk := connect.RecvMsg(conn, messages.File_Chunk_Request).(messages.File_chunk_response_t)

    log.Printf("%s: Recv File Chunk request for %s %d-th Chunk from %s\n", PeerName, filename, chunk_id, dest_peer_addr_str)
    
    /* Register every chunk once this peer gets */
    registerChunk(filename, chunk_id, PeerName)
    /*
    for {
        connect.SendChunkRegisterRequest(conn, filename, chunk_id, PeerName)
        chunk_register_resp := connect.RecvMsg(conn, messages.Chunk_Register_Request).(messages.Chunk_register_response_t)
        if chunk_register_resp.Succ == 1 {
            break
        }
    }
    /* Write the received chunk to local datastore */
    writeToLocalStore(filename, chunk_id, &(recv_chunk.Bytes))
    //dummyFunc2(recv_chunk.Bytes)
}

func downloadFile(filename string) {
    var wg sync.WaitGroup
    /* Get the file locations from the server first */
    //conn := connect.ConnectToServer("localhost", "9527")
    //conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1], PeerAddr[0], PeerAddr[1])
    conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1])

    /* Get every chunk according to the returned locations */
    connect.SendFileLocationsRequest(conn, filename)
    file_loc := connect.RecvMsg(conn, messages.File_Locations_Request).(messages.File_location_response_t)
    file_loc_info_arr := make([]file_loc_info_t, len(file_loc.Chunks_loc))
    
    log.Printf("%s: Gotten file locations with %d chunks\n", PeerName, len(file_loc.Chunks_loc))
    log.Println(PeerName, file_loc.Chunks_loc)

    for chunk_id := 0; chunk_id < len(file_loc.Chunks_loc); chunk_id++ {
        file_loc_info_arr[chunk_id].Chunk_id = chunk_id
        file_loc_info_arr[chunk_id].Num_locations = len(file_loc.Chunks_loc[chunk_id])
    }
    sort.Slice(file_loc_info_arr, func(i, j int) bool {
        return file_loc_info_arr[i].Num_locations < file_loc_info_arr[j].Num_locations
    })

    for _, info := range file_loc_info_arr {
        wg.Add(1)
        go recvChunk(&wg, filename, info.Chunk_id, file_loc.Chunks_loc[info.Chunk_id])
        //go dummyFunc(filename, info.Chunk_id, file_loc.Chunks_loc[info.Chunk_id])
    }
    
    wg.Wait()
    assembleChunks(filename, len(file_loc.Chunks_loc))
}

func shareFile(filename string) {
    /* Register the file */
    //conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1], PeerAddr[0], PeerAddr[1])
    conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1])
    file_info := getFileInfo(filename)
    if file_info == nil {
        return
    }
    log.Printf("%s: Share File %s, %d\n", PeerName, filename, file_info.Size())
    //connect.SendRegisterRequest(conn, []string{filename}, []int{1024})
    connect.SendRegisterRequest(conn, []string{filename}, []int{int(file_info.Size())}, PeerName)
    
    /* Receive the response */
    register_resp := connect.RecvMsg(conn, messages.Register_Request).(messages.Register_response_t)

    if register_resp.Register_succ[0] == 0 {
        log.Printf("%s: Register File %s failed!\n", PeerName, filename)
    } else {
        log.Printf("%s: Register File %s successfully!\n", PeerName, filename)
    }

    /* Register each chunk */
    num_chunks := (int(file_info.Size()) + (messages.CHUNK_SIZE-1)) / messages.CHUNK_SIZE
    //remain_size := file_info.Size()
    for i := 0; i < num_chunks; i++ {
        registerChunk(filename, i, PeerName)
    }
}

func getFileList() {
    log.Printf("%s: Get File List\n", PeerName)
    //conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1], PeerAddr[0], PeerAddr[1])
    conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1])
    connect.SendFileListRequest(conn)
    
    /* Receive the response */
    register_resp := connect.RecvMsg(conn, messages.Filelist_Request).(messages.Filelist_response_t)

    log.Println("Gotten the response of File List Request")
    log.Println(register_resp)
}

func processPeerRequest() {
    var line string
    scanner := bufio.NewScanner(os.Stdin)
    for {
        fmt.Println("------ Input the request ------")
        fmt.Println("----> Get-file-list")
        fmt.Println("----> Download <file name>")
        fmt.Println("----> Share <file name>")
        fmt.Println("")
        fmt.Println("")
        fmt.Print("$: ")
        //fmt.Scanln(&line)
        scanner.Scan()
        line = scanner.Text()
        input, file := parseUserInput(line)

        if input == DownloadFile {
            go downloadFile(file)
        } else if input == ShareFile {
            go shareFile(file)
        } else if input == GetFileList {
            go getFileList()
        }
    }
}

func main() {
    cmdArgs()
    PeerName = *ipAddr
    PeerAddr = strings.Split(PeerName, ":")
    go connect.RunServer(PeerAddr[0], PeerAddr[1])
    processPeerRequest()
    //_ = connect.RecvMsg(conn, 0)

}

