package ircutils

import (
	"regexp"
)

const (
	UNDEFINED = 000
	RPL_WELCOME = 001
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

		tempmap := make(map[string]string)
		temp.Fields = tempmap
		temp.Fields["client"] = temp.Prefix
		re := regexp.MustCompilePOSIX(":(.*) (.+)!(.+)@(.*)")

		if len(temp.Param) == 2 {
			// fmt.Println(re.MatchString(temp.Param[1]))
			// should fail if cannot match
			temp.Fields["recepient"] = temp.Param[0]
			matches := re.FindStringSubmatch(temp.Param[1])
			temp.Fields["message"]  = matches[1]
			temp.Fields["nick"]  = matches[2]
			temp.Fields["user"] = matches[3]
			temp.Fields["host"] = matches[4]
		}

		return temp
	
	default:
		temp := LexedMessage{}
		temp.Type = UNDEFINED
		temp.Prefix = m.Prefix
		temp.Command = m.Command
		temp.Param = m.Param
		temp.time = m.time
		return temp
	}
}
