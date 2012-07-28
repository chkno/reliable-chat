package main

import "testing"
import "time"

func TestMessageInsertAndRetreive(t *testing.T) {
	say := ""
	at := time.Now()
	var zero_time time.Time
	store := start_store()
	store.Add <- Message{at, say}
	messages_from_store := make(chan []Message, 1)
	store.Get <- StoreRequest{zero_time, messages_from_store}
	messages := <-messages_from_store
	if len(messages) != 1 {
		t.Fail()
	}
	if messages[0].Time != at {
		t.Fail()
	}
	if messages[0].Text != say {
		t.Fail()
	}
}
