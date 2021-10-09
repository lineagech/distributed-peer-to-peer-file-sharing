package main

import (
    _ "net"
    _ "log"
    "fmt"
    _ "bufio"
    "os"
    "flag"
    "connect"
    "messages"
    "strings"
    "sort"
)

type UserInput int


const (
    ShareFile UserInput = iota
    DownloadFile
    Invalid
)

type file_loc_info_t struct {
    Chunk_id int
    Num_locations int
}


type download_file struct {
    Chunks map[int][]byte
}

var downloadedFiles map[string]download_file

var fileDownloadPtr *string
var fileSharePtr *string
var dirPtr *string

func cmdArgs() {
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
    }
    split_line := strings.Split(line, ",")
    filename = split_line[1]

    return res, filename
}

func assembleChunks(filename string) {
    /* Get the file info (length) and the number of chunks the file has */

    /* Read one by one and write them to a single file */
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
    downloadedFiles[filename].Chunks[chunk_id] = *bytes_p
}

func recvChunk(filename string, chunk_id int, locations []string) {

    conn := connect.ConnectToServer("localhost", "9527")
    connect.SendFileChunkRequest(conn, filename, chunk_id)
    recv_chunk := connect.RecvMsg(conn, messages.File_Chunk_Request).(messages.File_chunk_response_t)
    
    /* Register every chunk once this peer gets */
    for {
        connect.SendChunkRegisterRequest(conn, filename, chunk_id)
        chunk_register_resp := connect.RecvMsg(conn, messages.Chunk_Register_Request).(messages.Chunk_register_response_t)
        if chunk_register_resp.Succ == 1 {
            break
        }
    }
    /* Write the received chunk to local datastore */
    writeToLocalStore(filename, chunk_id, &(recv_chunk.Bytes))
}

func downloadFile(filename string) {
    /* Get the file locations from the server first */
    conn := connect.ConnectToServer("localhost", "9527")

    /* Get every chunk according to the returned locations */
    connect.SendFileLocationsRequest(conn, filename)
    file_loc := connect.RecvMsg(conn, messages.File_Locations_Request).(messages.File_location_response_t)
    file_loc_info_arr := make([]file_loc_info_t, len(file_loc.Chunks_loc))
    for chunk_id := 0; chunk_id < len(file_loc.Chunks_loc); chunk_id++ {
        file_loc_info_arr[chunk_id].Chunk_id = chunk_id
        file_loc_info_arr[chunk_id].Num_locations = len(file_loc.Chunks_loc[chunk_id])
    }
    sort.Slice(file_loc_info_arr, func(i, j int) bool {
        return file_loc_info_arr[i].Num_locations < file_loc_info_arr[j].Num_locations
    })

    for _, info := range file_loc_info_arr {
        go recvChunk(filename, info.Chunk_id, file_loc.Chunks_loc[info.Chunk_id])
    }

}

func shareFile(filename string) {
    /* Register the file */
    conn := connect.ConnectToServer("localhost", "9527")
    file_info := getFileInfo(filename)
    if file_info == nil {
        return 
    }
    connect.SendRegisterRequest(conn, []string{"HelloWorld"}, []int{1024})

}

func processPeerRequest() {
    var line string
    for {
        fmt.Println("------ Input the request ------")
        fmt.Println("----> Download <file name>")
        fmt.Println("----> Share <file name>")
        fmt.Print("$: ")
        fmt.Scanln(&line)
        input, file := parseUserInput(line)

        if input == DownloadFile {
            go downloadFile(file)
        } else if input == ShareFile {
            go shareFile(file)
        }
    }
}

func main() {
    cmdArgs()
    processPeerRequest()
    //_ = connect.RecvMsg(conn, 0)

}

