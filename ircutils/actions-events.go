package ircutils

import (
	"fmt"
	"net"
	"os"
	"text/tabwriter"
	"github.com/fatih/color"
)

type Event struct {
	message ParsedMessage
	action func(h *IRCHandler) // need to change this to function
}

func NewEvent(m ParsedMessage) Event {
	var action func(h *IRCHandler)
	switch m.Command {
	case "001",  "003", "005", "253", "254", "255", "265", "266", "250":
		//RPL_WELCOME, RPL_CREATED,  RPL_ISUPPORT, RPL_LUSERUNKNOWN
		// RPL_LUSERCHANNELS, RPL_LUSERME, RPL_LOCALUSERS
		// RPL_GLOBALUSERS, RPL_LHIGHESTCONNCOUNT(?)
		action =  func(h *IRCHandler) {
			switch len(m.Param) {
			case 0:
				fmt.Fprintf(h.w, "\n")
			case 1:
				fmt.Fprintf(h.w, "%s\t\n", m.Prefix)

			case 2:
				fmt.Fprintf(h.w, "%s\t%s\t\n", m.Prefix, m.Param[1])
			case 3:
				fmt.Fprintf(h.w, "%s\t%s\t%s\t\n", m.Prefix, m.Param[1],
					m.Param[2])
			default:
				printstr := ""
				for i, j := range m.Param {
					if i == 0 {
						continue
					}
					printstr += j+" "
				}
				printstr += "\n"
				fmt.Fprintf(h.w, printstr)
			}
			h.w.Flush()
		}
	case "002", "004", "MODE": // RPL_YOURHOST, RPL_MYINFO, MODE
		action =  func(h *IRCHandler) {
			color.Set(color.FgRed)
			for i, j := range m.Param {
				if i == 0 { // first parameter is the nickname
					continue
				}
				//fmt.Println(m.Command, j)
				fmt.Println(j)
			}
			color.Unset()
		}
	case "251", "252": // RPL_LUSERCLIENT, RPL_LUSEROP
		action =  func(h *IRCHandler) {
			color.Set(color.FgYellow)
			for i, j := range m.Param {
				if i == 0 { // first parameter is the nickname
					continue
				}
				//fmt.Println(m.Command, j)
				fmt.Println(j)
				
			}
			color.Unset()
		}
	case "NOTICE", "375", "376": //RPL_MOTDSTART
		action = func(h *IRCHandler) {
			color.Set(color.FgGreen)
			color.Set(color.Italic)
			for i, j := range m.Param {
				if i == 0  { // first parameter is the nickname
					continue
				}
				//fmt.Println(m.Command, j)
				fmt.Println(j)
			}

			color.Unset()
		}
	case "372" : // MOTD
		action = func(h *IRCHandler) {
			color.Set(color.BgBlack)
			color.Set(color.FgHiCyan)
			for i, j := range m.Param { 
				if i < 1 { // first two parameters are nick and client
					continue
				}
				//fmt.Println(m.Command, j)
				fmt.Println(j)
			}
			color.Unset()
		}
//	case "376": // RPL_ENDOFMOTD
//		break

	case "PING":
		action = func(h *IRCHandler) {
			h.conn.Write([]byte("PONG "+":"+m.Prefix+"\r\n"))
		}
	default:
		action = func(h *IRCHandler) {
			fmt.Println("Unrecognized command: ", m.Command)
			fmt.Println("Prefix: ", m.Prefix)
			fmt.Println("Command: ", m.Command)
			for i, j := range m.Param {
				fmt.Printf("Param %d: %s ", i, j)
			}
		}
	}
	return Event{message: m, action: action}

	
}

type IRCHandler struct {
	conn net.Conn
	w *tabwriter.Writer
	// maybe add mesages
}

func NewHandler(conn net.Conn) IRCHandler {
	return IRCHandler{conn: conn,
		w: tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)}
}

func (h *IRCHandler) Act(e Event) {
	e.action(h)
}
