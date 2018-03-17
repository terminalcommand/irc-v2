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

func elasticPrettyPrint(p LexedMessage) {
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
	var action func(h *IRCHandler)
	switch m.Type {
	// instead of distinguishing between different messages at event dispatch
	// I'm planning a lexing stage, where parsed messages will be categorized into their
		// own data types, I will then use the type as the identifier
	case RPL_WELCOME:
		action = func(h *IRCHandler) {
			elasticPrettyPrint(m)
		}
	case NOTICE:
		action = func(h *IRCHandler) {
			elasticPrettyPrint(m)
		}
	case UNDEFINED:
		action = func(h *IRCHandler) {
			elasticPrettyPrint(m)
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
	e.action(h)
}
