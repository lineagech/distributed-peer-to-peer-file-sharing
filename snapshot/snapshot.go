package snapshot

import (
    "os"
    "fmt"
    "log"
    "strings"
    "bytes"
    _ "io"
    "strconv"
    _ "sort"
    _ "encoding/json"
    "bufio"
    "encoding/gob"
    "sync"
)

type fileSnapshot struct {
    Fd *os.File
    FileLen int
    //File_mtx sync.Mutex
}

const (
    CHUNK_SIZE int = 1024*1024
)

var fileChunkList map[string][]int
var snapshotName = "snapshot.status"
var ssfile *os.File = nil
var downloadingFileList []string
var downloadedChunks = make(map[string]fileSnapshot)
var fileList_mtx sync.Mutex
var write_mtx sync.Mutex
var fileSnapshot_mtx sync.Mutex

func ReadFromSnapshot(filename string) []int {
    f, err := os.Open(filename+"."+snapshotName) 
    if err != nil {
        log.Printf("Open %s failed!\n", filename+"."+snapshotName)
        return nil
    }
    var file_len int
    var num_chunks int
    buf_reader := bufio.NewReader(f)

    line, err := buf_reader.ReadBytes('\n')
    file_len, _ = strconv.Atoi(string(line[:len(line)-1]))
    num_chunks = (file_len + (CHUNK_SIZE-1)) / CHUNK_SIZE
    
    if num_chunks == 0 {
        return nil
    }

    var ret = make([]int, num_chunks)
    for i := 0; i < num_chunks; i++ {
        ret[i] = 0
    }
    for {
        line, err = buf_reader.ReadBytes('\n')
        if err != nil {
            break;
        }
        id, _ := strconv.Atoi(string(line[:len(line)-1]))
        if id >= num_chunks {
            return nil
        }
        ret[id] = 1
    }
    f.Close()

    // Re-Construct single file snapshot
    for i := 0; i < len(ret); i++ {
        if ret[i] == 1 {
            WriteFileSnapshot(filename, file_len, i)
        }
    }

    return ret
}

func WriteFileSnapshot(filename string, file_len int, chunk_id int) {
    fileSnapshot_mtx.Lock()
    _, found := downloadedChunks[filename]
    if found != true {
        // Create snapshot for a single file
        fd, _ := os.Create(filename+"."+snapshotName)
        downloadedChunks[filename] = fileSnapshot {
            fd,
            file_len,
        }

        fd.WriteString(strconv.Itoa(file_len)+"\n")
    }
    // Record which chunks has been downloaded...
    //downloadedChunks[filename].File_mtx.Lock()
    downloadedChunks[filename].Fd.WriteString(strconv.Itoa(chunk_id)+"\n")
    //downloadedChunks[filename].File_mtx.Unlock()
    fileSnapshot_mtx.Unlock()
}

func ReadSnapshot() []string {
    f, err := os.Open(snapshotName)
    if err != nil {
        log.Printf("Open %s failed!\n", snapshotName)
        return nil
    }
    var ret []string
    buf_reader := bufio.NewReader(f)
    for {
        line, err := buf_reader.ReadBytes('\n')
        if err != nil {
            break;
        }
        filename := string(line[:len(line)-1])
        log.Println("Read from global snapshot: ", filename)
        ret = append(ret, filename)
    }
    f.Close()

    for _, filename := range ret {
        RegisterFile(filename)
    }
    return ret
}

func WriteSnapshot() {
    write_mtx.Lock()
    defer write_mtx.Unlock()
    for _, f := range downloadingFileList {
        ssfile.WriteString(f+"\n")
    }
}

func RegisterFile(filename string) {
    fileList_mtx.Lock()
    if ssfile == nil {
        // Create snapshot for global status
        ssfile, _ = os.Create(snapshotName)
        //ssfile, _ = os.Open(snapshotName)
    }

    // Check if already exists...
    for _, s := range downloadingFileList {
        if strings.Compare(s, filename) == 0{
            return
        }
    }

    downloadingFileList = append(downloadingFileList, filename)
    fmt.Println("Downloading List: ", downloadingFileList)
    WriteSnapshot()
    fileList_mtx.Unlock()
}

func FinishDownload(filename string) {
    fileList_mtx.Lock()
    defer fileList_mtx.Unlock()
    var fi = -1
    for i := 0; i < len(downloadingFileList); i++ {
        if strings.Compare(filename, downloadingFileList[i]) == 0 {
            fi = i
            break
        }
    }
    if fi != -1 {
        downloadingFileList[fi] = downloadingFileList[len(downloadingFileList)-1]
        downloadingFileList = downloadingFileList[:len(downloadingFileList)-1]
    }
    ssfile.Truncate(0)
    //ssfile.Close()
    //ssfile = nil
    WriteSnapshot()
    os.Remove(filename+"."+snapshotName)
}

func AddDumpFileChunk(filename string, chunk_id int) {
    _ = os.Getpid()
    chunk_list := fileChunkList[filename]

    /* insert to sorted chunk-id array */
    var insert_index = len(chunk_list)
    for i, e := range chunk_list {
        if e > chunk_id {
            insert_index = i
            break
        }
    }

    fileChunkList[filename] = append(chunk_list[:insert_index+1], chunk_list[insert_index:]...)
    
    /* convert to byte array */
    buffer := new(bytes.Buffer)
    e := gob.NewEncoder(buffer)

    _ = e.Encode(fileChunkList)

    /* dump to a file */
    f, _ := os.Create(snapshotName)
    f.Write(buffer.Bytes())
    
    defer f.Close()
}
