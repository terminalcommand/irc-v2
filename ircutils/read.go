package ircutils

import (
	"bufio"
	"bytes"
)

// Copied and pasted from stackoverflow, beware!
// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func ScanCRLF(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte{'\r', '\n'}); i >= 0 {
		// We have a full newline-terminated line.
		return i + 2, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil}


// my code starts again
func ReadLine(reader *bufio.Reader) (string, error) {
//	scanner := bufio.NewScanner(reader)
//	scanner.Split(ScanCRLF)
	
//	scanner.Scan()
	line, err := reader.ReadString('\n')
//	line, err := r.ReadLine() // Looks for \r\n and \n
//	next ,err := reader.Peek(1)
//	for next[0] != byte('\n') {
//		temp, _ := reader.ReadString('\r')
//		line += temp
//		next ,err = reader.Peek(1)
//	}
	return line, err
}
