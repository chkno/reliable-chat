package main

import "testing"
import "runtime"
import "time"

func TestMessageInsertAndRetreive(t *testing.T) {
	say := "'Ello, Mister Polly Parrot!"
	at := time.Now()
	var zero_time time.Time
	store := start_store()
	store.Add <- &Message{at, say}
	messages_from_store := make(chan []Message, 1)
	store.Get <- &StoreRequest{zero_time, messages_from_store}
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
	close(store.Get)
	close(store.Add)
}

func TestFetchBlocksUntilSpeak(t *testing.T) {
	start_fetch_wait_count := fetch_wait_count.String()
	say := "I've got a lovely fresh cuttle fish for you"
	at := time.Now()
	var zero_time time.Time
	store := start_store()
	messages_from_store := make(chan []Message, 1)
	store.Get <- &StoreRequest{zero_time, messages_from_store}
	for start_fetch_wait_count == fetch_wait_count.String() {
		runtime.Gosched()
	}
	store.Add <- &Message{at, say}
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
	close(store.Get)
	close(store.Add)
}
