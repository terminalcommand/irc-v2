package ircutils

// https://web.stanford.edu/class/archive/cs/cs143/cs143.1128/

import (
	"log"
	"bufio"
	"io"
	"strings"
	"unicode"
	"time"
)

type ParsedMessage struct {
	Raw string
	Prefix string
	Command string
	Param []string
	time time.Time
}

func ParseMsg(m Message) ParsedMessage {
	parsed := Parse(m.GetString())
	parsed.Raw = m.GetString()
	parsed.time = m.GetTime()
	return parsed
}

func Parse(s string) ParsedMessage {
	var cursor = bufio.NewReader(strings.NewReader(s))
	var parsedPrefix bool
	var parsedCommand bool

	var result ParsedMessage

	for {
		nextc, _, err := cursor.ReadRune()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal("Error scanning the next character", err)
		}
	switch {
	case nextc==':':
		if !parsedPrefix {
			//log.Println("Prefix incoming ")
			result.Prefix = ParsePrefix(cursor)
			parsedPrefix = true
		} //else {
		//	log.Println("Parameter incoming ")
		//	ParseParameter(cursor)
		//}
	case nextc=='@':
		if !parsedPrefix {
			log.Println("Tag incoming ")
			// Not implemented yet
		}
	case unicode.IsDigit(nextc) || unicode.IsLetter(nextc):
		if !parsedCommand {
			cursor.UnreadRune()
			//log.Println("Command incoming ")
			result.Command = ParseCommand(cursor)
			parsedCommand = true
		} else if parsedPrefix {
			cursor.UnreadRune()
			result.Param = append(result.Param, ParseParameter(cursor)...)
		}

	case nextc==' ':
		//fmt.Println(cursor.ReadRune())
		continue
	default:
		if !parsedPrefix {
			log.Fatal("There is no Prefix in the IRC Message")
		} else if !parsedCommand {
			log.Fatal("There is no Command in the IRC Message")
		} else { // when all else fails treat is a parameter? For example params starting with *
			// log.Println("Parameter incoming")
			cursor.UnreadRune()
			result.Param = append(result.Param, ParseParameter(cursor)...)
		}
	}
	}
	return result // the end result as ParsedMessage
}
// ParsePrefix starts parsing after the initial : (colon), starting : (colon) is thereby ommited in the result
func ParsePrefix(cursor *bufio.Reader) string {
	var buffer string
	for {
		nextc, _, err := cursor.ReadRune()
		if strings.ContainsRune(" ", nextc) {
			break
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal("Error parsing prefix ", err)
		}
		buffer += string(nextc)
	}
	//fmt.Printf("Prefix:\t\t%s \n", buffer)
	return buffer
}

func ParseCommand(cursor *bufio.Reader) string {
	var buffer string
	for {
		nextc, _, err := cursor.ReadRune()
		if strings.ContainsRune(" ", nextc) {
			break
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal("Error parsing command ", err)
		}
		buffer += string(nextc)
	}
	// fmt.Printf("Command:\t\t%s \n", buffer)
	return buffer
}

// ParseParameter results  include the starting : (colon) if provided
func ParseParameter(cursor *bufio.Reader) []string {
	var buffer []string
	buffer = make([]string, 1)
	var paramCount = 0
	for {
		nextc, _, err := cursor.ReadRune()
		//fmt.Println(nextc)
		if strings.ContainsRune(" ", nextc) {
			if strings.ContainsRune(":", rune(buffer[paramCount][0])) {
				// parameter starting with : can have spaces in it
				buffer[paramCount] += string(nextc)
				continue
			}

			paramCount++
			var tmp = make([]string, paramCount+1)
			copy(tmp, buffer)
			buffer = tmp
			continue
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal("Error parsing command ", err)
		}
		//fmt.Println(paramCount)
		buffer[paramCount] += string(nextc)
	}
	//for i, j := range buffer {
	//	fmt.Printf("Parameter %d : %s\n", i+1, j)
	//}
	return buffer
}

func Example() {
	var message = ":bar.example.com 001 amy :Welcome to the Internet Relay Network borja!borja@polaris.cs.uchicago.edu"
	message = message + string([]byte{'\r', '\n'})
//	log.Println(message)
	Parse(message)

	message = ":bar.example.com 433 * amy :Nickname is already in use."
	message = message + string([]byte{'\r', '\n'})
	Parse(message)

}

