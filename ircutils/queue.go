package ircutils

import (
	"container/list"
	"time"
	"log"
)
// embeds the doubly linked list from container/list
type MessageList struct {
	*list.List
}

type Message struct {
	text string
	time time.Time
}

func (m Message) GetString() string {
	return m.text
}

func (m Message) GetTime() time.Time {
	return m.time
}


func NewMessage(m interface{}) (Message) {
	return Message{text: m.(string), time: time.Now()}
}

func NewMessageList() MessageList {
	return MessageList{list.New()}
}

func (l *MessageList) Poll() ([]Message) {
	// Poll returns the whole messagelist
	var ml []Message
	for e := l.Front(); e != nil; e = e.Next() {
		ml = append(ml, e.Value.(Message)) // ugly hack, I admit :(
	}
	return ml
}


func (l *MessageList) PollNew(last Message) ([]Message) {
	// PollNew returns the new message since lastMessage
	// If lastMessage is nil or not in MessageList returns the whole list
	// Warning: currently extremely unstable!
	var ml []Message
	var searchIndex *list.Element
	for e := l.Back(); e!=nil; e = e.Prev() {
		searchIndex = e
		if e.Value == last {
			// if we reached the last message break
			break
		}
		log.Println("did not find a matching message")
		// if we can't find search index the loop will set searchindex to the first element
		// so it returns the whole message list, if it can't find the last message
		// it is safe to use PollNew instead of Poll
	}
	for e := searchIndex; e!=nil; e = e.Next() {
		if e == searchIndex && e.Next()==nil {
			// there is no new value so do not return anything
			break
		}
		ml = append(ml, e.Value.(Message))
	}
	return ml
}

func (l *MessageList) PollLast() (Message) {
	var m Message
	e := l.Back()
	if e != nil {
		m = e.Value.(Message)
	}
	return m
}
