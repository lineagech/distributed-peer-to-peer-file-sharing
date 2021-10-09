package connect

import (
    "fmt"
    "net"
    "os"
    "bufio"
    "log"
    "messages"
    _ "encoding/json"
    "time"
)

func TestMsg() string {
    return "Connect"
}

func SendMsg(conn net.Conn, msg []byte) {
    log.Println("Send msg ", string(msg))
    conn.Write([]byte(msg))
    //re, _ := bufio.NewReader(conn).ReadString('\n')
    //log.Println("Server relay: ", re)
}

func RecvMsg(conn net.Conn, req messages.PeerRequest) interface{} {
    log.Printf("RecvMsg (%s)", conn.RemoteAddr().String())
    msg, _ := bufio.NewReader(conn).ReadBytes('\n')

    res := messages.ParseResponse(req, msg[0:len(msg)-1])
    //log.Printf("RecvMsg (%s) %v\n", conn.RemoteAddr().String(), res.(messages.Register_response_t))
    return res
}

//func ConnectToServer(dest_ip string, dest_port string, src_ip string, src_port string) net.Conn {
func ConnectToServer(dest_ip string, dest_port string) net.Conn {
    fmt.Println("Connecting to ", dest_ip+":"+dest_port)
    //server, _ := net.ResolveTCPAddr("tcp", dest_ip+":"+dest_port)
    //client, _ := net.ResolveTCPAddr("tcp", src_ip+":"+src_port)
    var conn net.Conn
    for {
        c, err := net.Dial("tcp", dest_ip+":"+dest_port)
        //c, err := net.DialTCP("tcp", client, server)
        if err != nil {
            fmt.Println("Error connecting:", err.Error())
            time.Sleep(5 * time.Second)
            //os.Exit(1)
        } else {
            conn = c
            break
        }
    }

    //conn.Write([]byte("God...\n"))

    return conn
}

func SendRegisterRequest(conn net.Conn, files []string, lengths []int, peer_addr string) {
    fmt.Println("Send Register Request ", conn.RemoteAddr().String(), files, lengths)
    req := messages.EncodeRegisterRequest(files, lengths, peer_addr)
    SendMsg(conn, req)
}

func SendFileListRequest(conn net.Conn) {
    fmt.Println("Send File List Request ", conn.RemoteAddr().String())
    req := messages.EncodeFileListRequest()
    SendMsg(conn, req)
}

func SendFileLocationsRequest(conn net.Conn, filename string) {
    fmt.Println("Send File Locations Request for %s", conn.RemoteAddr().String(), filename)
    req := messages.EncodeFileLocationsRequest(filename)
    SendMsg(conn, req)
}

func SendChunkRegisterRequest(conn net.Conn, filename string, chunk_id int, peer_addr string) {
    fmt.Println("Send Chunk Register Request", conn.RemoteAddr().String())
    req := messages.EncodeChunkRegisterRequest(filename, chunk_id, peer_addr)
    SendMsg(conn, req)
}

func SendFileChunkRequest(conn net.Conn, filename string, chunk_id int) {
    fmt.Println("Send File Chunk Request", conn.RemoteAddr().String())
    req := messages.EncodeFileChunkRequest(filename, chunk_id)
    SendMsg(conn, req)
}


func RunServer(ip string, port string) {
    /* tcp connection */
    fmt.Println("Create server at " + ip + ":" + port)
    ln, err := net.Listen("tcp", ip+":"+port)
    if err != nil {
        fmt.Println("Error listening:", err.Error());
        os.Exit(1);
    }

    defer ln.Close()
    
    for {
        c, err := ln.Accept()
        if err != nil {
            fmt.Println("Error connecting:", err.Error())
            return
        }
        fmt.Println("Peer " + c.RemoteAddr().String() + " connected.")

        // handle connection concurrently
        go handleConnection(c)
    }
}

func handleConnection(conn net.Conn) {
    buffer, err := bufio.NewReader(conn).ReadBytes('\n')

    if err != nil {
        fmt.Println("Client left.")
        conn.Close()
        return
    }
    
    //for i := 0; i < len(buffer); i++ {
    //    log.Printf("%d : %c\n", i, buffer[i])
    //}
    // log is safe for multi-threading
    log.Println("Client message:", string(buffer[:len(buffer)-1]), conn.RemoteAddr().String())
    
    // handle the reqeust
    response := messages.HandlePeerRequest(buffer, conn.RemoteAddr().String())

    // Send response message to the client
    conn.Write(response)
    
    // Restart the process
    go handleConnection(conn)
    
}
