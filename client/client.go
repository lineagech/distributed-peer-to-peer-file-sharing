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
    "snapshot"
    "golang.org/x/sys/unix"
    "io/ioutil"
)

type UserInput int
var PeerName string
var ServerAddr  = []string {"127.0.0.1", "9527"}
var PeerAddr []string
var FileList messages.Filelist_response_t
 
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
var downloadedProgress = make(map[string][]bool)
var downloadProgress_mtx sync.Mutex
var downloadFiles_mtx sync.Mutex

var pendingChunks = make(map[string][]bool)

var ipAddr *string
var fileDownloadPtr *string
var fileSharePtr *string
var dirPtr *string
var resumePtr *bool

var filePrefix = make(map[string]string)

func dummyFunc(filename string, chunk_id int, locations []string) {
    log.Printf("%s: %s, %d\n", PeerName, filename, chunk_id)
    log.Println(locations)
}

func dummyFunc2(bytes []byte) {
    log.Println("Dummy Recv Bytes ", len(bytes))
}

func getConsoleSize(fd int) (width, height int, err error) {
    ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ);
    if err != nil {
        return -1, -1, err
    }
    return int(ws.Col), int(ws.Row), nil
}

func getDownloadProgress(filename string) int {
    cnt := 0;
    downloadProgress_mtx.Lock()
    progress := downloadedProgress[filename]
    downloadProgress_mtx.Unlock()
    for i := 0; i < len(progress); i++ {
        if progress[i] == true {
            cnt++
        }
    }
    percent := float32(cnt)/float32(len(progress))

    return int(percent*100)
}

func printProgress(p int) {
    width, _, _ := getConsoleSize(0)
    width = width*4/5
    var line = ""
    fmt.Printf("\r")
    for j := 0; j < int(width*p/100.0); j++ {
        line += "="
    }
    line += ">>> "+ strconv.Itoa(p)+"%% Downloaded..."
    fmt.Printf(line)
}

func showDownloadProgress(filename string) {
    fmt.Printf("\n %s: \n", filename)
    for {
        var p = getDownloadProgress(filename)
        printProgress(p)
        if p == 100 {
            break
        }
    }
    fmt.Println("\n")
    fmt.Println("")
}

func cmdArgs() {
    ipAddr = flag.String("ip", "", "ip exposed to the network")
    fileDownloadPtr = flag.String("download", "", "file to download")
    fileSharePtr = flag.String("file-to-share", "", "file to share")
    dirPtr = flag.String("dir-to-share", "", "directory to share")
    resumePtr = flag.Bool("resume", false, "Resume")
    flag.Parse()
}

func getDirPath(filename string) string {
    var ret string
    _, found := filePrefix[filename]
    if found != true {
        return ""
    }
    ret = filePrefix[filename]+"/"
    return ret
}

func getFilesToShare() []string {
    var ret []string
    if *fileSharePtr != "" {
        ret = append(ret, *fileSharePtr)
    }
    if *dirPtr != "" {
        // get dir contents 
        files, _ := ioutil.ReadDir(*dirPtr)
        for _, file := range files {
            if file.IsDir() == false {
                ret = append(ret, file.Name())
                filePrefix[file.Name()] = *dirPtr
            }
        }
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
    //log.Println("Line ", line)
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
    for i := 0; i < num_chunks; i++ {
        file.Write(downloadedFiles[filename].Chunks[i])
    }
    file.Sync()
}

func getFileInfo(filename string) os.FileInfo {
    file_info, err := os.Stat(filename)
    if err != nil {
        file_info, err = os.Stat(getDirPath(filename)+filename)
        if err != nil {
            fmt.Println("get file info error!", filename)
            return nil
        }
    }
    return file_info
}

func getFileLenFromList(filename string) int {
    for i, name := range FileList.Filename {
        if strings.Compare(filename, name) == 0 {
            return FileList.Length[i]
        }
    }
    return 0
}

func updateDownloadProgress(filename string, chunk_id int) {
    downloadProgress_mtx.Lock()
    defer downloadProgress_mtx.Unlock()
    nowLen := len(downloadedProgress[filename])
    if chunk_id >= nowLen {
        tmp := make([]bool, chunk_id+1)
        copy(tmp, downloadedProgress[filename])
        downloadedProgress[filename] = tmp
    }
    downloadedProgress[filename][chunk_id] = true
}

func readFromLocalStore(filename string, chunk_id int) {
    _, found := downloadedFiles[filename]
    if found != true {
        downloadedFiles[filename] = download_file_t {
            make(map[int][]byte),
        }
    }
    chunk_filename := strconv.Itoa(chunk_id) + "_" + filename;
    downloadedFiles[filename].Chunks[chunk_id], _ = os.ReadFile(chunk_filename)
}

func writeToLocalStore(filename string, chunk_id int, bytes_p *[]byte) {
    downloadFiles_mtx.Lock()
    _, found := downloadedFiles[filename]
    if found != true {
        downloadedFiles[filename] = download_file_t {
            make(map[int][]byte),
        }
    }
    downloadedFiles[filename].Chunks[chunk_id] = *bytes_p
    downloadFiles_mtx.Unlock()

    /* Write to the file */
    chunk_filename := strconv.Itoa(chunk_id) + "_" + filename;
    //log.Printf("%s: Write recv bytes to %s\n", PeerName, chunk_filename)
    err := os.WriteFile(chunk_filename, *bytes_p, 0644)
    if err != nil {
        fmt.Println("Chunk File Write Error!")
    }
    //downloadedProgress[filename][chunk_id] = true
    updateDownloadProgress(filename, chunk_id)
}

func registerChunk(filename string, chunk_id int, peer_addr string) {
    for {
        conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1])
        err := connect.SendChunkRegisterRequest(conn, filename, chunk_id, PeerName)
        if err != nil {
            conn.Close()
            continue
        }
        recv_msg, err := connect.RecvMsg(conn, messages.Chunk_Register_Request)
        if err != nil {
            conn.Close()
            continue
        }
        chunk_register_resp := recv_msg.(messages.Chunk_register_response_t)

        if chunk_register_resp.Succ == 1 {
            //log.Printf("%s: Register Chunk for %s %d-th chunk Succeeded\n", PeerName, filename, chunk_id)
            break
        } else {
            //log.Printf("%s: Register Chunk for %s %d-th chunk Failed\n", PeerName, filename, chunk_id)
            continue
        }
    }
}

func recvChunk(wg *sync.WaitGroup, filename string, chunk_id int, locations []string) {
    defer wg.Done()
    for {
        rand_index := rand.Int()
        dest_peer_addr_str := locations[rand_index % len(locations)]
        if strings.Compare(dest_peer_addr_str, PeerName) == 0 {
            rand_index++
            dest_peer_addr_str = locations[rand_index % len(locations)]
        }

        dest_peer_addr := strings.Split(dest_peer_addr_str, ":")

        //conn := connect.ConnectToServer(dest_peer_addr[0], dest_peer_addr[1], PeerAddr[0], PeerAddr[1])
        conn := connect.ConnectToServer(dest_peer_addr[0], dest_peer_addr[1])
        err := connect.SendFileChunkRequest(conn, filename, chunk_id)
        if err != nil {
            conn.Close()
            continue
        }

        //log.Printf("%s: Send File Chunk request for %s %d-th Chunk to %s\n", PeerName, filename, chunk_id, dest_peer_addr_str)
        recv_msg, err := connect.RecvMsg(conn, messages.File_Chunk_Request)
        if err != nil {
            conn.Close()
            continue
        }
        recv_chunk := recv_msg.(messages.File_chunk_response_t)

        //log.Printf("%s: Recv File Chunk request for %s %d-th Chunk from %s\n", PeerName, filename, chunk_id, dest_peer_addr_str)

        /* Register every chunk once this peer gets */
        registerChunk(filename, chunk_id, PeerName)

        /* Write the received chunk to local datastore */
        writeToLocalStore(filename, chunk_id, &(recv_chunk.Bytes))
        //dummyFunc2(recv_chunk.Bytes)
        break
    }
    /* Do snapshot */
    var file_len = getFileLenFromList(filename)
    snapshot.WriteFileSnapshot(filename, file_len, chunk_id)
}

func downloadMissingChunks(filename string) {
    var wg sync.WaitGroup

    /* Get the file locations from the server first */
    conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1])

    /* Get every chunk according to the returned locations */
    _ = connect.SendFileLocationsRequest(conn, filename)
    recv_msg, err := connect.RecvMsg(conn, messages.File_Locations_Request)
    if err != nil {
        fmt.Println("Download Missing Chunks connection error!")
    }
    file_loc := recv_msg.(messages.File_location_response_t)
    file_loc_info_arr := make([]file_loc_info_t, len(file_loc.Chunks_loc))

    //log.Printf("%s: Gotten file locations with %d chunks\n", PeerName, len(file_loc.Chunks_loc))
    //log.Println(PeerName, file_loc.Chunks_loc)

    for chunk_id := 0; chunk_id < len(file_loc.Chunks_loc); chunk_id++ {
        file_loc_info_arr[chunk_id].Chunk_id = chunk_id
        file_loc_info_arr[chunk_id].Num_locations = len(file_loc.Chunks_loc[chunk_id])
    }
    sort.Slice(file_loc_info_arr, func(i, j int) bool {
        return file_loc_info_arr[i].Num_locations < file_loc_info_arr[j].Num_locations
    })

    for _, info := range file_loc_info_arr {
        if pendingChunks[filename][info.Chunk_id] == true {
            wg.Add(1)
            go recvChunk(&wg, filename, info.Chunk_id, file_loc.Chunks_loc[info.Chunk_id])
        }
    }

    /* Show the progress */
    showDownloadProgress(filename)

    wg.Wait()
    assembleChunks(filename, len(file_loc.Chunks_loc))
    /* Remove from the snapshot*/
    snapshot.FinishDownload(filename)
}

func downloadFile(filename string) {
    var wg sync.WaitGroup
    for {
        /* Do snapshot globally*/ 
        snapshot.RegisterFile(filename)

        /* Get the file locations from the server first */
        conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1])

        /* Get every chunk according to the returned locations */
        err := connect.SendFileLocationsRequest(conn, filename)
        if err != nil {
            conn.Close()
            continue
        }
        recv_msg, err := connect.RecvMsg(conn, messages.File_Locations_Request)
        if err != nil {
            conn.Close()
            continue
        }
        file_loc := recv_msg.(messages.File_location_response_t)
        if len(file_loc.Chunks_loc) == 0 {
            fmt.Println("Cannot Donwload --- ", filename)
            return
        }

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
        }

        /* Show the progress */
        showDownloadProgress(filename)

        wg.Wait()
        assembleChunks(filename, len(file_loc.Chunks_loc))
        break
    }
    /* Remove from the snapshot*/
    snapshot.FinishDownload(filename)
}

func shareFile(filename string) {
    for {
        /* Register the file */
        //conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1], PeerAddr[0], PeerAddr[1])
        conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1])
        file_info := getFileInfo(filename)
        if file_info == nil {
            return
        }
        //log.Printf("%s: Share File %s, %d\n", PeerName, filename, file_info.Size())
        //connect.SendRegisterRequest(conn, []string{filename}, []int{1024})
        err := connect.SendRegisterRequest(conn, []string{filename}, []int{int(file_info.Size())}, PeerName)
        if err != nil {
            conn.Close()
            continue
        }

        /* Receive the response */
        recv_msg, err := connect.RecvMsg(conn, messages.Register_Request)
        if err != nil {
            conn.Close()
            continue
        }
        register_resp := recv_msg.(messages.Register_response_t)

        if register_resp.Register_succ[0] == 0 {
            log.Printf("%s: Register File %s failed!\n", PeerName, filename)
            return
        } else {
            log.Printf("%s: Register File %s successfully!\n", PeerName, filename)
        }

        /* Register each chunk */
        num_chunks := (int(file_info.Size()) + (messages.CHUNK_SIZE-1)) / messages.CHUNK_SIZE
        for i := 0; i < num_chunks; i++ {
            registerChunk(filename, i, PeerName)
        }
        break
    }
}

func getFileList() {
    //log.Printf("%s: Get File List\n", PeerName)
    //conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1], PeerAddr[0], PeerAddr[1])
    conn := connect.ConnectToServer(ServerAddr[0], ServerAddr[1])
    _ = connect.SendFileListRequest(conn)

    /* Receive the response */
    recv_msg, err := connect.RecvMsg(conn, messages.Filelist_Request)
    if err != nil {
        fmt.Println("Get file list connection error!")
    }
    register_resp := recv_msg.(messages.Filelist_response_t)
    FileList = register_resp

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
        if len(line) == 0 {
            continue
        }
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

func isContaining(list []string, element string) bool {
    for _, s := range list {
        if strings.Compare(s, element) == 0 {
            return true
        }
    }
    return false
}

func resumeProcess() {
    Ungoing_files := snapshot.ReadSnapshot()
    var Complete_missing_files []string
    for _, filename := range Ungoing_files {
        chunk_status := snapshot.ReadFromSnapshot(filename)
        if chunk_status == nil {
            Complete_missing_files = append(Complete_missing_files, filename)
            continue
        }
        pendingChunks[filename] = make([]bool, len(chunk_status))
        for chunk_id, status := range chunk_status {
            if status == 1 {
                readFromLocalStore(filename, chunk_id)
                updateDownloadProgress(filename, chunk_id)
                pendingChunks[filename][chunk_id] = false
            } else {
                // update to resume downloading data structures
                pendingChunks[filename][chunk_id] = true
            }
        }
    }

    for _, filename := range Ungoing_files {
        if isContaining(Complete_missing_files, filename) == false {
            downloadMissingChunks(filename)
        }
    }

    for _, filename := range Complete_missing_files {
        downloadFile(filename)
    }

    fmt.Println("\n----------------------- Resume Process Done ----------------------\n")
}

func shareUserFiles() {
    messages.RegisterDirPathCallBack(getDirPath)
    files := getFilesToShare()
    if len(files) != 0 {
        for _, filename := range files {
            shareFile(filename)
        }
    }
}

func main() {
    cmdArgs()
    if len(*ipAddr) == 0 {
        os.Exit(-1)
    }
    PeerName = *ipAddr
    PeerAddr = strings.Split(PeerName, ":")
    if *resumePtr == true {
        resumeProcess()
    }
    go connect.RunServer(PeerAddr[0], PeerAddr[1])
    shareUserFiles()
    processPeerRequest()
    //_ = connect.RecvMsg(conn, 0)

}

