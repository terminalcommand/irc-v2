package main

import (
	"irc-v2/ircutils"
	"log"
	"bufio"
)

func main() {
	conn, err := ircutils.Connect("chat.freenode.net", "6667")
	if err != nil {
		log.Fatal("Unable to connect ", err)
	}
	defer conn.Close()

	messages := ircutils.NewMessageList() // message store for all incoming msgs
	quitp := make(chan bool) // channel to signal SendServer (tcp server for input) to stop
	go ircutils.SendServer(conn, quitp) // start the tcp server to listen and forward irc commands
	defer func() {quitp <- true } () // set quitp to true at the end of the program to stop SendServer

	// Login commands
	conn.Write([]byte("NICK terminaltest\r\n")) 
	conn.Write([]byte("USER terminaltest * 8 : terminal test\r\n"))
	conn.Write([]byte("JOIN #haskell\r\n"))

	// Setting up the reader and the main program loop
	reader := bufio.NewReader(conn) // reader for incoming messages, must be outside the loop
	irchandler := ircutils.NewHandler(conn) // custom handler with embedding -- not implemented yet
	for {
		line, err := ircutils.ReadLine(reader)
		if err != nil {
			log.Fatal("Error Reading Line ", err)
		}
		messages.PushBack(ircutils.NewMessage(line)) // ReadLine and push it
		// Parse the last incoming message and act on it
		parsedm := ircutils.ParseMsg(messages.PollLast())
		lexedm := ircutils.LexMsg(parsedm)
		event := ircutils.NewEvent(lexedm) // Maybe DispatchMsg
		irchandler.Act(event)
		}
}


