package ircutils

import (
	"regexp"
	"log"
)

const (
	UNDEFINED = 000
	RPL_WELCOME = 001
	RPL_TOPIC = 332
	NOTICE = iota
	PING = iota
	PRIVMSG = iota
)

type LexedMessage struct {
	ParsedMessage
	Type int
	Fields map[string]string
}

func (m *ParsedMessage) initParsedMsg(t int) LexedMessage {
	// converts a ParsedMessage into a LexedMessage
	// initializes a new LexedMessage
	// copies all values from the ParsedMessage into the
	// newly initialized LexedMessage
	temp := LexedMessage{}
	temp.Type = t
	temp.Prefix = m.Prefix
	temp.Command = m.Command
	temp.Param = m.Param
	temp.time = m.time
	temp.Raw = m.Raw
	
	tempmap := make(map[string]string)
	temp.Fields = tempmap

	return temp
}

func LexMsg(m ParsedMessage) LexedMessage {
	switch m.Command {
	// https://defs.ircdocs.horse/defs/numerics.html
	case "001":
		// RPL_WELCOME
		// <client> :Welcome to the Internet Relay Network <nick>!<user>@<host>
		temp := m.initParsedMsg(RPL_WELCOME)

   		temp.Fields["server"] = temp.Prefix
		relong := regexp.MustCompilePOSIX(":(.*) (.+)!(.+)@(.*)")
		reshort := regexp.MustCompilePOSIX(":(.*) (.+)") 

		if len(temp.Param) == 2 {
			matchedlong := relong.MatchString(temp.Param[1])
			matchedshort := reshort.MatchString(temp.Param[1])
			if matchedlong {
				temp.Fields["client"] = temp.Param[0]
				matches := relong.FindStringSubmatch(temp.Param[1])
				temp.Fields["message"]  = matches[1]
				temp.Fields["nick"]  = matches[2]
				temp.Fields["user"] = matches[3]
				temp.Fields["host"] = matches[4]

			} else if matchedshort {
				temp.Fields["client"] = temp.Param[0]
				matches := reshort.FindStringSubmatch(temp.Param[1])
				temp.Fields["message"] = matches[1]
				temp.Fields["nick"] = matches[2]
			} else {
				log.Fatal("No match found, faulty RPL_WELCOME message ", temp.Param[1])
			}
		}

		return temp

	case "NOTICE":
		//NOTICE
		temp := m.initParsedMsg(NOTICE)

		temp.Fields["server"] = temp.Prefix
		re := regexp.MustCompilePOSIX(":(.*)") // using a regex here may be an overkill
		
		if len(temp.Param) == 2 {
			temp.Fields["target"] = temp.Param[0]
			matches := re.FindStringSubmatch(temp.Param[1])
			temp.Fields["text"] = matches[1]
		}

		return temp

	case "PING":
		temp := m.initParsedMsg(PING)
		temp.Fields["server"] = temp.Prefix

		return temp

	case "PRIVMSG":
		//<target>{,<target>} <text to be sent>
		// targets split with , are not considered seperately
		temp := m.initParsedMsg(PRIVMSG)

		// used the same code from NOTICE
		temp.Fields["server"] = temp.Prefix
		re := regexp.MustCompilePOSIX(":(.*)") // using a regex here may be an overkill
		
		if len(temp.Param) == 2 {
			temp.Fields["target"] = temp.Param[0]
			matches := re.FindStringSubmatch(temp.Param[1])
			temp.Fields["text"] = matches[1]
		}

		return temp

	case "332":
		// RPL_TOPIC
		temp := m.initParsedMsg(RPL_TOPIC)
		// used the same code from NOTICE and PRIVMSG, only field names are different
		temp.Fields["server"] = temp.Prefix

		re := regexp.MustCompilePOSIX(":(.*)") // using a regex here may be an overkill
		
		if len(temp.Param) == 3 {
			temp.Fields["client"] = temp.Param[0]
			temp.Fields["channel"] = temp.Param[1]
			matches := re.FindStringSubmatch(temp.Param[2])
			temp.Fields["topic"] = matches[1]
		}

		return temp

	default:
		temp := m.initParsedMsg(UNDEFINED)
		return temp
	}
}

