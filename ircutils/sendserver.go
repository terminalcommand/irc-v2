package ircutils

import (
	"net"
	"bufio"
	"log"
)

func SendServer(conn net.Conn, quitp chan bool) {
	log.Println("Server listening on port 8081")
	ln, err := net.Listen("tcp", ":8081")
	if err != nil {
		log.Fatal("Cannot open a local port on 8081")
	}

	// listen on all interfaces
	// accept a connection on port
	inConn, err := ln.Accept()
	if err != nil {
		log.Fatal("Cannot accept incoming connection request")
	}

	reader := bufio.NewReader(inConn)
	for {
		select {
		case <- quitp:
			return
		default:
			message, _ := reader.ReadString('\n')
			//log.Println(message)
			if message!="" {
				conn.Write([]byte(message + "\r\n"))
			}
		}
	}
}
