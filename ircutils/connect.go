package ircutils

import (
	"net"
	"strings"
)

func Connect(hostname string, port string) (net.Conn, error){
	conn, err := net.Dial("tcp", strings.Join([]string{hostname, port}, ":"))
	return conn, err
}
