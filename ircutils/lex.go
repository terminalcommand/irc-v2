package ircutils

import (
	"regexp"
	"log"
)

const (
	UNDEFINED = 000
	RPL_WELCOME = 001
	NOTICE = iota
)

type LexedMessage struct {
	ParsedMessage
	Type int
	Fields map[string]string
}

func LexMsg(m ParsedMessage) LexedMessage {
	switch m.Command {
	case "001":
		//RPL_WELCOME
		temp := LexedMessage{}
		temp.Type = RPL_WELCOME
		temp.Prefix = m.Prefix
		temp.Command = m.Command
		temp.Param = m.Param
		temp.time = m.time
		temp.Raw = m.Raw

		tempmap := make(map[string]string)
		temp.Fields = tempmap
		temp.Fields["client"] = temp.Prefix
		relong := regexp.MustCompilePOSIX(":(.*) (.+)!(.+)@(.*)")
		reshort := regexp.MustCompilePOSIX(":(.*) (.+)") 

		if len(temp.Param) == 2 {
			matchedlong := relong.MatchString(temp.Param[1])
			matchedshort := reshort.MatchString(temp.Param[1])
			if matchedlong {
				temp.Fields["recepient"] = temp.Param[0]
				matches := relong.FindStringSubmatch(temp.Param[1])
				temp.Fields["message"]  = matches[1]
				temp.Fields["nick"]  = matches[2]
				temp.Fields["user"] = matches[3]
				temp.Fields["host"] = matches[4]

			} else if matchedshort {
				temp.Fields["recepient"] = temp.Param[0]
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
		temp := LexedMessage{}
		temp.Type = NOTICE
		temp.Prefix = m.Prefix
		temp.Command = m.Command
		temp.Param = m.Param
		temp.time = m.time
		temp.Raw = m.Raw

		tempmap := make(map[string]string)
		temp.Fields = tempmap
		temp.Fields["client"] = temp.Prefix
		re := regexp.MustCompilePOSIX(":(.*)") // using a regex here may be an overkill
		
		if len(temp.Param) == 2 {
			temp.Fields["target"] = temp.Param[0]
			matches := re.FindStringSubmatch(temp.Param[1])
			temp.Fields["text"] = matches[1]
		}

		return temp
	
	default:
		temp := LexedMessage{}
		temp.Type = UNDEFINED
		temp.Prefix = m.Prefix
		temp.Command = m.Command
		temp.Param = m.Param
		temp.time = m.time
		temp.Raw = m.Raw
		return temp
	}
}
