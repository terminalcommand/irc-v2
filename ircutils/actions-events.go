package ircutils

import (
	"fmt"
	"net"
	"os"
	"text/tabwriter"
	"log"
)

type Event struct {
	message interface{}
	action func(h *IRCHandler) // need to change this to function
}

func logToConsole(p LexedMessage) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.StripEscape)
	fmt.Fprintln(w, "Type\t:", p.Type)
	fmt.Fprintln(w, "Prefix\t:", p.Prefix)
	fmt.Fprintln(w, "Command\t:", p.Command)
	for i, j := range p.Param {
		fmt.Fprintln(w, "Param ", i, " \t:", j)
	}
	for i, j := range p.Fields {
		fmt.Fprintln(w, i, "\t:", j)
	}
	w.Flush()
}
func printMessage(client string, text string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.StripEscape)
	fmt.Fprintln(w, client+"\t ", text)
	w.Flush()
}
func logToFile(p LexedMessage) {
	f, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	w := tabwriter.NewWriter(f, 0, 0, 1, ' ', tabwriter.StripEscape)
	fmt.Fprintln(w, "Raw\t:", p.Raw)
	fmt.Fprintln(w, "Type\t:", p.Type)
	fmt.Fprintln(w, "Prefix\t:", p.Prefix)
	fmt.Fprintln(w, "Command\t:", p.Command)
	for i, j := range p.Param {
		fmt.Fprintln(w, "Param ", i, " \t:", j)
	}

	for i, j := range p.Fields {
		fmt.Fprintln(w, i, "\t:", j)
	}
	w.Flush()
}


func NewEvent(m LexedMessage) Event {
	// feel free to write up your own NewEvent function or change this function
	// to add/change the behaviour of your IRC Client
	var action func(h *IRCHandler)
	switch m.Type {
	// instead of distinguishing between different messages at event dispatch
	// I'm planning a lexing stage, where parsed messages will be categorized into their
	// own data types, I will then use the type as the identifier
	// Edit I think I've had accomplished this. Comment above doesn't make sense now :)
	case RPL_WELCOME:
		action = func(h *IRCHandler) {
			printMessage(m.Fields["server"], m.Fields["message"])
		}
	case NOTICE:
		action = func(h *IRCHandler) {
			printMessage(m.Fields["server"], m.Fields["text"])
		}
	case PING:
		action = func(h *IRCHandler) {
			//elasticPrettyPrint(m)
			h.conn.Write([]byte("PONG :"+m.Fields["server"]))
		}
	case PRIVMSG:
		action = func(h *IRCHandler) {
			printMessage(m.Fields["server"], m.Fields["text"])	
		}
	case RPL_TOPIC:
		action = func(h *IRCHandler) {
			printMessage(m.Fields["channel"], m.Fields["topic"])	
		}

	
	case UNDEFINED:
		action = func(h *IRCHandler) {
			logToFile(m)
		}
	}

	return Event{message: m, action: action}
}

type IRCHandler struct {
	conn net.Conn
}

func NewHandler(conn net.Conn) IRCHandler {
	return IRCHandler{conn: conn}
}

func (h *IRCHandler) Act(e Event) {
	if e.action != nil {
		e.action(h)
	} else {
		fmt.Println("Encountered a strange error: action of an event is nil")
	}
}
