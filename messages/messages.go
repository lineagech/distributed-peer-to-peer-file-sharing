package messages

import (
    "os"
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
    "log"
    _ "time"
    _ "path/filepath"
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
    CHUNK_SIZE int = 1024*1024
)

/* Internal Structs */
type file_info_t struct {
    Filename string
    Length   int
    PeerAddr string
}

type file_loc_t struct {
    Chunks_loc map[int][]string
}

/* Request and Response Definition */
type Register_request_t struct {
    Num_files int
    Filename []string
    Length   []int
    PeerAddr string
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

type Chunk_register_request_t struct {
    Filename string
    Chunk_id int
    PeerAddr string
}

type Chunk_register_response_t struct {
    Succ int
}

type File_chunk_request_t struct {
    Filename string
    Chunk_id int
}

type File_chunk_response_t struct {
    Bytes []byte
}

var PeerRequestStr = [...]string{ "Register Request",
                                  "File List Request",
                                  "File Locations Request",
                                  "Chunk Register Request",
                                  "File Chunk Request" }
/* Status */
var Character string

/* Shared Resources */
var snapshotCallback func (string,int) = nil
var fileLocSnapshotCallback func (string, int, []string) = nil
var dirPathCallback func(string)string = nil
var fileList = list.New()
var fileLocMap = map[string]file_loc_t {

}
var registerTable = map[string]interface{}{
    "" : nil,
}
/*------------------------------------------*/

/* Peer Resources */

/*------------------------------------------*/

func max(x, y int) int {
    if x < y {
        return y
    }
    return x
}

func check_err(err error) {
    if err != nil {
        fmt.Println(err.Error())
        panic(err)
    }
}

func local_chunk_name(filename string, chunk_id int) string {
    chunk_name := filename + "_" + strconv.Itoa(chunk_id)
    return chunk_name
}

func printFileList() {
    fmt.Println("Print File List -------------------------------- ")
    for e := fileList.Front(); e != nil; e = e.Next() {
        var info = (*e).Value.(file_info_t)
        fmt.Println("\t", info)
    }
    fmt.Println("=================================================")
}

func get_file_length(filename string) int {
    for e := fileList.Front(); e != nil; e = e.Next() {
        info := e.Value.(file_info_t)
        if strings.Compare(info.Filename, filename) == 0 {
            return info.Length
        }
    }
    return 0
}

func insertToFileList(filename_str string, length int, peer_addr string) {
    file_info := file_info_t {
        filename_str,
        length,
        peer_addr,
    }
    var inserted = false
    if fileList.Len() == 0 {
        fileList.PushBack(file_info)
    } else {
        for e := fileList.Front(); e != nil; e = e.Next() {
            if strings.Compare((*e).Value.(file_info_t).Filename, filename_str) == 1 {
                fileList.InsertBefore(file_info, e)
                inserted = true
                break
            }
        }
        if inserted == false {
            fileList.PushBack(file_info)
        }
    }
}

func handle_register(msg []byte, peerAddr string) Register_response_t {
    //fmt.Println("Handle register request from peer ", peerAddr)
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
    req := Register_request_t{}
    json.Unmarshal(msg, &req)
    //fmt.Println("Request Content: ", req)

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

        //fmt.Println("File ", filename_str, ", Length ", length)
        insertToFileList(filename_str, length, req.PeerAddr)
        response.Register_succ[i] = 1

        // snaphsot
        if snapshotCallback != nil {
            snapshotCallback(filename_str, length)
        }
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
        i += 1
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

func handle_chunk_register(msg []byte) Chunk_register_response_t {
    var req Chunk_register_request_t 
    var res Chunk_register_response_t

    json.Unmarshal(msg, &req)

    res = register_file_chunk(req.PeerAddr, req.Filename, req.Chunk_id)

    return res
}

func handle_file_chunk(msg []byte) File_chunk_response_t {
    var req File_chunk_request_t

    json.Unmarshal(msg, &req)

    return get_file_chunk_locally(req.Filename, req.Chunk_id)
}

func get_file_locations(filename string) File_location_response_t{
    var res_loc = File_location_response_t {
        make(map[int][]string),
    }
    _, found := fileLocMap[filename]
    if found == false {
        return res_loc
    }

    //log.Printf("Get File Locations for %s\n", filename)
    file_loc := fileLocMap[filename]
    num_chunks := (get_file_length(filename)+(CHUNK_SIZE-1)) / CHUNK_SIZE
    
    //log.Printf("Number of Chunks %d\n", num_chunks)
    for i := 0; i < num_chunks; i++ {
       res_loc.Chunks_loc[i] = file_loc.Chunks_loc[i]
    }
    return res_loc
}

func register_file_chunk(peer_addr string, filename string, chunk_id int) Chunk_register_response_t {
    /* Check if in fileLocMap for this filename */
    _, found := fileLocMap[filename]
    if found != true {
        fileLocMap[filename] = file_loc_t {
            make(map[int][]string),
        }
    }

    //log.Printf("register file chunk for %s (%s, %d-th chunk)\n", peer_addr, filename, chunk_id)

    // check the chunk is valid or not
    if (chunk_id < 0) {
        return Chunk_register_response_t{Succ: 0}
    }
    num_chunks := (get_file_length(filename)+(CHUNK_SIZE-1)) / CHUNK_SIZE
    if (chunk_id >= num_chunks) {
        return Chunk_register_response_t{Succ: 0}
    }

    /* Check chunk id exists or not */
    _, found = fileLocMap[filename].Chunks_loc[chunk_id]
    if found != true {
        fileLocMap[filename].Chunks_loc[chunk_id] = make([]string, 0)
    }

    file_loc := fileLocMap[filename]

    // check if already registered
    for _, s := range file_loc.Chunks_loc[chunk_id] {
        // found 
        if (strings.Compare(s, peer_addr) == 0) {
            return Chunk_register_response_t{Succ: 1}
        }
    }

    //log.Printf("Insert peer addr for %s (%s, %d-th chunk)\n", peer_addr, filename, chunk_id)

    // if not, insert the peer address
    file_loc.Chunks_loc[chunk_id] = append(file_loc.Chunks_loc[chunk_id], peer_addr)

    // snaphost
    if fileLocSnapshotCallback != nil {
        fileLocSnapshotCallback(filename, chunk_id, file_loc.Chunks_loc[chunk_id])
    }

    return Chunk_register_response_t{Succ: 1}
}

/*
func get_src_file_chunk(filename string, chunk_id) File_chunk_response_t {
   if _, err := os.Stat(filename); os.IsNotExist(err) {
        log.Printf("File %s not exist!\n", filename)
        return File_chunk_response_t{}
    } 
    if (chunk_id < 0) {
        return File_chunk_response_t{}
    }
    file_length := get_file_length(filename)
    num_chunks := (file_length+(CHUNK_SIZE-1)) / CHUNK_SIZE
    if (chunk_id >= num_chunks) {
        return File_chunk_response_t{}
    }
    
    file, _ := os.Open(filename)
    defer file.Close()
    
    new_pos, err := file.Seek()
}
*/

func getFileInfo(filename string) os.FileInfo {
    file_info, err := os.Stat(filename)
    if err != nil {
        if dirPathCallback != nil {
            file_info, err = os.Stat(dirPathCallback(filename)+filename)
        }
        if err != nil {
            fmt.Println("get file info error!", filename)
            return nil
        }
    }
    return file_info
}


func get_file_chunk_locally(filename string, chunk_id int) File_chunk_response_t {
    //file_loc := fileLocMap[filename]
    
    // check the chunk is valid or not
    if (chunk_id < 0) {
        log.Printf("get file chunk locally for %s: chunk id %d < 0!\n", filename, chunk_id)
        return File_chunk_response_t{}
    }

    chunk_filename := local_chunk_name(filename, chunk_id)
    var file_length int
    var read_length int
    //file_length := get_file_length(filename)
    //num_chunks := (file_length+(CHUNK_SIZE-1)) / CHUNK_SIZE
    //if (chunk_id >= num_chunks) {
    //    log.Printf("get file chunk locally for %s: chunk id %d >= num chunks %d!\n", filename, chunk_id, num_chunks)
    //    return File_chunk_response_t{}
    //}
    var file_info = getFileInfo(filename)
    var complete_file_exists = false
    if file_info == nil {
        file_info = getFileInfo(chunk_filename)
        read_length = int(file_info.Size())
    } else {
        // calcualate the length for read
        complete_file_exists = true
        file_length = int(file_info.Size())
        read_length = CHUNK_SIZE
        if (chunk_id+1)*CHUNK_SIZE > file_length {
            read_length = file_length - chunk_id*CHUNK_SIZE
        }
    }

    //log.Printf("Read Length for (%s, %d-th chunk) would be %d\n", filename, chunk_id, read_length)

    var res = File_chunk_response_t {
        Bytes: make([]byte, read_length),
    }

    // read from the local file
    //abs_path, _ := filepath.Abs("./"+filename)
    if complete_file_exists {
        //log.Printf("%s exists locally.\n", filename)
        if dirPathCallback != nil {
            filename = dirPathCallback(filename) + filename
        }
        f, err := os.Open(filename)
        check_err(err)

        // move to desired location, seek: 0->from origin, 1->from current position
        //log.Printf("%s seek locally.\n", filename)
        _, err = f.Seek(int64(chunk_id)*int64(CHUNK_SIZE), 0)
        check_err(err)

        //log.Printf("%s read locally.\n", filename)
        _, err = f.Read(res.Bytes)
        check_err(err)
        //fmt.Println("Read Content for %s (chunk id %d): %s\n", filename, chunk_id, string(res.Bytes))
    }

    //log.Println(string(res.Bytes))

    /*
    f, err := os.Open(chunk_name)
    check_err(err)
    
    // move to desired location, seek: 0->from origin, 1->from current position
    _, err = f.Seek(int64(chunk_id)*int64(CHUNK_SIZE), 0)
    check_err(err)

    _, err = f.Read(res.Bytes)
    check_err(err)
    fmt.Println("Read Content for %s (chunk id %d): %s\n", filename, chunk_id, string(res.Bytes))
    */

    return res
}

func HandlePeerRequest(msg []byte, peerAddr string) []byte {
    var req PeerRequest
    
    msg_reader := bytes.NewBuffer(msg)
    s, _ := msg_reader.ReadBytes(';')
    i, _ := strconv.ParseInt(string(s[0:len(s)-1]), 10, 32)
    req = PeerRequest(i)
    //log.Println("Handle Peer Request ", req)
    msg, _ = msg_reader.ReadBytes('\n');

    switch req {
    case Register_Request:
        fmt.Println("Handle Peer Request: Register Request", string(msg))
        res := handle_register(msg, peerAddr)
        res_byte, _ := json.Marshal(&res)
        //fmt.Println("Response for the register request ", string(res_byte))
        return []byte(string(res_byte)+"\n")
    case Filelist_Request:
        fmt.Println("Handle Peer Request: File List Request")
        res := handle_filelist()
        //res := handle_file_list()
        res_byte, _ := json.Marshal(res)
        //fmt.Println("Response for the file list request", string(res_byte))
        return []byte(string(res_byte)+"\n")
    case File_Locations_Request:
        fmt.Println("Handle Peer Request: File Locations Request")
        res := handle_file_locations(msg)
        res_byte, _ := json.Marshal(&res)
        //fmt.Println("Response for the file locations request", string(res_byte))
        return []byte(string(res_byte)+"\n")
    case Chunk_Register_Request:
        fmt.Println("Handle Peer Request: Chunk Register Request")
        res := handle_chunk_register(msg)
        res_byte, _ := json.Marshal(&res)
        //fmt.Println("Response for the chunk register request", string(res_byte))
        return []byte(string(res_byte)+"\n")
    case File_Chunk_Request:
        fmt.Println("Handle Peer Request: File Chunk Request")
        res := handle_file_chunk(msg)
        res_byte, _ := json.Marshal(&res)
        //fmt.Println("Response for the file chunk request", string(res_byte))
        return []byte(string(res_byte)+"\n")
    default:
        break
    }
    return nil
}

func EncodeRegisterRequest(files []string, lengths []int, peer_addr string) []byte {
    var req = Register_request_t {
        len(files),
        files,
        lengths,
        peer_addr,
    }
    req_bytes, _ := json.Marshal(&req)
    //fmt.Println("Encode Register Request: ", string(req_bytes))
    
    var req_type PeerRequest = Register_Request
    var req_str string
    req_str = strconv.Itoa(int(req_type)) + ";" + string(req_bytes) + "\n"
    
    //fmt.Println("Encode Register Request: ", req_str)

    return []byte(req_str)
}

func EncodeFileListRequest() []byte {
    var req_type PeerRequest = Filelist_Request
    var req_str string
    req_str = strconv.Itoa(int(req_type)) + ";" + "\n"
    //fmt.Println("Encode File List Request: ", req_str)
    return []byte(req_str)
}

func EncodeFileLocationsRequest(filename string) []byte {
    var req_type PeerRequest = File_Locations_Request
    var req_str string
    var req = File_location_request_t {
        filename,
    }
    req_bytes, _ := json.Marshal(&req)
    req_str = strconv.Itoa(int(req_type)) + ";" + string(req_bytes) + "\n"
    //fmt.Println("Encode File Locations Request: ", req_str)
    return []byte(req_str)
}

func EncodeChunkRegisterRequest(filename string, chunk_id int, peer_addr string) []byte {
    var req_type PeerRequest = Chunk_Register_Request
    var req_str string
    var req = Chunk_register_request_t {
        filename,
        chunk_id,
        peer_addr,
    }
    req_bytes, _ := json.Marshal(&req)
    req_str = strconv.Itoa(int(req_type)) + ";" + string(req_bytes) + "\n"
    //fmt.Println("Encode Chunk Register Request: ", req_str)
    return []byte(req_str)
}

func EncodeFileChunkRequest(filename string, chunk_id int) []byte {
    var req_type PeerRequest = File_Chunk_Request
    var req_str string
    var req = File_chunk_request_t {
        filename,
        chunk_id,
    }
    req_bytes, _ := json.Marshal(&req)
    req_str = strconv.Itoa(int(req_type)) + ";" + string(req_bytes) + "\n"
    //fmt.Println("Encode File Chunk Request: ", req_str)
    return []byte(req_str)
}

func ParseResponse(req PeerRequest, msg []byte) interface{} {
    switch req {
    case Register_Request:
        //fmt.Println("Handle Peer Response: Register Request Response", string(msg))
        var response Register_response_t
        json.Unmarshal(msg, &response)
        //fmt.Println("Response content: ", response)
        return response
    case Filelist_Request:
        //fmt.Println("Handle Peer Response: File List Response", string(msg))
        var response Filelist_response_t
        json.Unmarshal(msg, &response)
        //fmt.Println("Response content: ", response)
        return response
    case File_Locations_Request:
        //fmt.Println("Handle Peer Response: File Locations Response", string(msg))
        var response File_location_response_t
        json.Unmarshal(msg, &response)
        //fmt.Println("Response content: ", response)
        return response
    case Chunk_Register_Request:
        //fmt.Println("Handle Peer Response: Chunk Register Response", string(msg))
        var response Chunk_register_response_t
        json.Unmarshal(msg, &response)
        //fmt.Println("Response content: ", response)
        return response
    case File_Chunk_Request:
        //fmt.Println("Handle Peer Response: File Chunk Response", string(msg))
        var response File_chunk_response_t
        json.Unmarshal(msg, &response)
        //fmt.Println("Response content: ", response)
        return response
    default:
        break

    }
    return nil
}

func RegisterSnapshotCallBack(callback func(string, int)){
    snapshotCallback = callback
}

func RegisterFileLocSnapshotCallBack(callback func(string, int, []string)) {
    fileLocSnapshotCallback = callback
}

func RegisterDirPathCallBack(callback func(string)string) {
    dirPathCallback = callback
}

func SetP2PServerFileList(file_list []string, length []int) {
    for i, filename := range file_list {
        insertToFileList(filename, length[i], "")
    }
    printFileList()
}

func SetFileLoc(filename string, loc map[int][]string) {
    fileLocMap[filename] = file_loc_t {
        //make(map[int][]string),
        loc,
    }
    fmt.Println("Set File Locations for ", filename)
    fmt.Println(loc)
    //fileLocMap[filename].Chunks_loc = loc
}
