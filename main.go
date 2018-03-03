package main

import (
	"irc-v2/ircutils"
	"log"
	"bufio"
)

func main() {
	conn, err := ircutils.Connect("chat.freenode.net", "6667")
	defer conn.Close()
	if err != nil {
		log.Fatal("Unable to connect ", err)
	}

	messages := ircutils.NewMessageList() // message store for all incoming msgs
	quitp := make(chan bool) // channel to signal SendServer (tcp server for input) to stop
	go ircutils.SendServer(conn, quitp) // start the tcp server to listen and forward irc commands
	defer func() {quitp <- true } () // set quitp to true at the end of the program to stop SendServer

	// Login commands
	conn.Write([]byte("NICK terminaltest\r\n")) 
	conn.Write([]byte("USER terminaltest * 8 : terminal test\r\n"))

	// Setting up the reader and the main program loop
	reader := bufio.NewReader(conn) // reader for incoming messages, must be outside the loop
	irchandler := ircutils.NewHandler(conn)
	for {
		line, err := ircutils.ReadLine(reader)
		messages.PushBack(ircutils.NewMessage(line)) // ReadLine and push it
		if err != nil {
			log.Fatal("Error Reading Line ", err)
		}
		// Parse the last incoming message and act on it
		irchandler.Act(ircutils.NewEvent(
			ircutils.Parse(messages.PollLast().GetString())))
		}
}


