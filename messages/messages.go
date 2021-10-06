package messages

import (
    "fmt"
    _ "net"
    "container/list"
    "strings"
    "bytes"
    _ "io"
    "strconv"
    _ "sort"
    //"lab1_responses"
    "encoding/json"
    _ "bufio"
)

type PeerRequest int

const (
    Register_Request PeerRequest = iota
    Filelist_Request
    File_Locations_Request
    Chunk_Register_Request
    File_Chunk_Request
)

const (
    CHUNK_SIZE int = 1024
)

/* Internal Structs */
type file_info_t struct {
    Filename string
    Length   int
}

type file_loc_t struct {
    Chunks_loc map[int][]string
}

/* Request and Response Definition */
type register_request_t struct {
    Num_files int
    Filename []string
    Length   []int
}

type Register_response_t struct {
    //register_succ interface{}
    Register_succ []int
}

type Filelist_request_t struct {
    // Nothing to specify
}

type Filelist_response_t struct {
    Num_files int
    Filename []string
    Length []int
}

type File_location_request_t struct {
    Filename string
}

type File_location_response_t struct {
    /* N-th chunk : [ip addr...]*/
    Chunks_loc map[int][]string
}

var PeerRequestStr = [...]string{ "Register Request",
                                  "File List Request",
                                  "File Locations Request",
                                  "Chunk Register Request",
                                  "File Chunk Request" }
/* Shared Resources */
var fileList = list.New()
var fileLocMap = make(map[string]file_loc_t)
var registerTable = map[string]interface{}{
    "" : nil,
}
/********************/



func printFileList() {
    fmt.Println("Print File List")
    for e := fileList.Front(); e != nil; e = e.Next() {
        var info = (*e).Value.(file_info_t)
        fmt.Println("\t", info)
    }
}

func get_file_length(filename string) int {
    for e := fileList.Front(); e != nil; e = e.Next() {
        info := e.Value.(file_info_t)
        if strings.Compare(info.Filename, filename) == 1 {
            return info.Length
        }
    }
    return 0
}

func insertToFileList(filename_str string, length int) {
    file_info := file_info_t {
        filename_str,
        length,
    }
    if fileList.Len() == 0 {
        fileList.PushBack(file_info)
    } else {
        for e := fileList.Front(); e != nil; e = e.Next() {
            if strings.Compare((*e).Value.(file_info_t).Filename, filename_str) == 1 {
                fileList.InsertBefore(file_info_t{filename_str, length,}, e)
            }
        }
    }
}

func handle_register(msg []byte, peerAddr string) Register_response_t {
    fmt.Println("Handle register request from peer ", peerAddr)
    _, found := registerTable[peerAddr]
    if found != true {
        registerTable[peerAddr] = map[string]int{}
    }
    /* Parse the msg
    --- number of files, <filename1, length1>, .....
    --- delimieter is ';'
    */
    //msg_reader := bytes.NewBuffer(msg)
    //num_files_str, err := msg_reader.ReadBytes(';')
    //if err != nil {
    //    fmt.Println(err.Error())
    //    return nil
    //}
    //num_files, err := strconv.ParseInt(string(num_files_str), 10, 64)
    req := register_request_t{}
    json.Unmarshal(msg, &req)
    fmt.Println("Request Content: ", req)

    //num_files, _ := strconv.ParseInt(string(req.Num_files), 10, 64)
    num_files := req.Num_files

    //var filename, length []byte
    //var filename_str string
    //var response = make([]byte, num_files, num_files)

    //var t [num_files]int
    var t = make([]int, num_files)
    response := Register_response_t{t}

    for i := 0; i < int(num_files); i++ {
        //
        //filename, err = msg_reader.ReadBytes(';')
        //if err != nil {
        //    fmt.Println(err.Error())
        //    return response
        //}
        //filename_str = string(filename)
        filename_str := req.Filename[i]

        //length, err = msg_reader.ReadBytes(';')
        // Search the position of the list to insert
        //if fileList.Len() == 0 {
        //    fileList.PushBack(filename_str)
        //    continue
        //}
        //length, _ := strconv.ParseInt(req.length[i], 10, 64)
        length := req.Length[i]

        fmt.Println("File ", filename_str, ", Length ", length)
        insertToFileList(filename_str, length)
        response.Register_succ[i] = 1
    }
    
    printFileList()
    return response
}


func handle_filelist() Filelist_response_t {
    var res = Filelist_response_t {
        fileList.Len(),
        make([]string, fileList.Len()),
        make([]int, fileList.Len()),
    }
    
    var i = 0
    for e := fileList.Front(); e != nil; e = e.Next() {
        res.Filename[i] = (*e).Value.(file_info_t).Filename
        res.Length[i] = (*e).Value.(file_info_t).Length
    }

    return res
}

func handle_file_locations(msg []byte) File_location_response_t {
    var req File_location_request_t
    var res File_location_response_t
 
    json.Unmarshal(msg, &req)
 
    res = get_file_locations(req.Filename)
    
    return res
}

func get_file_locations(filename string) File_location_response_t{
    var res_loc File_location_response_t 
    
    file_loc := fileLocMap[filename]
    num_chunks := get_file_length(filename) / CHUNK_SIZE
    
    for i := 0; i < num_chunks; i++ {
       res_loc.Chunks_loc[i] = file_loc.Chunks_loc[i]
    }
    return res_loc
}

func HandlePeerRequest(msg []byte, peerAddr string) []byte {
    var req PeerRequest
    
    msg_reader := bytes.NewBuffer(msg)
    s, _ := msg_reader.ReadBytes(';')
    i, _ := strconv.ParseInt(string(s), 10, 32)
    req = PeerRequest(i)
    msg, _ = msg_reader.ReadBytes('\n');

    switch req {
    case Register_Request:
        fmt.Println("Handle Peer Request: Register Request", string(msg))
        res := handle_register(msg, peerAddr)
        res_byte, _ := json.Marshal(&res)
        fmt.Println("Response for the register request ", string(res_byte))
        return []byte(string(res_byte)+"\n")
    case Filelist_Request:
        fmt.Println("Handle Peer Request: File List Request")
        res := handle_filelist()
        //res := handle_file_list()
        res_byte, _ := json.Marshal(res)
        fmt.Println("Response for the file list request", string(res_byte))
        return []byte(string(res_byte)+"\n")
    case File_Locations_Request:
        fmt.Println("Handle Peer Request: File Locations Request")
        res := handle_file_locations(msg)
        res_byte, _ := json.Marshal(&res)
        fmt.Println("Response for the file locations request", string(res_byte))
        return []byte(string(res_byte)+"\n")
    case Chunk_Register_Request:
        fmt.Println("Handle Peer Request: Chunk Register Request")
        //handle_chunk_register()
        break
    case File_Chunk_Request:
        fmt.Println("Handle Peer Request: File Chunk Request")
        //handle_file_chunk()
        break
    default:
        break
    }
    return nil
}

func EncodeRegisterRequest(files []string, lengths []int) []byte {
    var req = register_request_t {
        len(files),
        files,
        lengths,
    }
    req_bytes, _ := json.Marshal(&req)
    fmt.Println("Encode Register Request: ", string(req_bytes))
    
    var req_type PeerRequest = Register_Request
    var req_str string
    req_str = strconv.Itoa(int(req_type)) + ";" + string(req_bytes) + "\n"
    
    fmt.Println("Encode Register Request: ", req_str)

    return []byte(req_str)
}

func EncodeFileListRequest() []byte {
    var req_type PeerRequest = Filelist_Request
    var req_str string
    req_str = strconv.Itoa(int(req_type)) + ";" + "\n"

    fmt.Println("Encode Register Request: ", req_str)

    return []byte(req_str)
}

func ParseResponse(req PeerRequest, msg []byte) interface{} {
    switch req {
    case Register_Request:
        fmt.Println("Handle Peer Response: Register Request Response", string(msg))
        var response Register_response_t
        json.Unmarshal(msg, &response)
        fmt.Println("Response content: ", response)
        return response
    case Filelist_Request:
        var response Filelist_response_t
        json.Unmarshal(msg, &response)
        fmt.Println("Response content: ", response)
        return response
    }
    return nil
}
