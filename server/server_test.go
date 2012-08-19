/*  reliable-chat - multipath chat
 *  Copyright (C) 2012  Scott Worley <sworley@chkno.net>
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU Affero General Public License as
 *  published by the Free Software Foundation, either version 3 of the
 *  License, or (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU Affero General Public License for more details.
 *
 *  You should have received a copy of the GNU Affero General Public License
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import "testing"
import "runtime"
import "strconv"
import "time"

func expectMessage(t *testing.T, m *Message, at time.Time, id, say string) {
	if m.Time != at {
		t.Fail()
	}
	if m.ID != id {
		t.Fail()
	}
	if m.Text != say {
		t.Fail()
	}
}

func TestMessageInsertAndRetreive(t *testing.T) {
	id := "1"
	say := "'Ello, Mister Polly Parrot!"
	at := time.Now()
	var zero_time time.Time
	store := start_store()
	store.Add <- &Message{at, id, say}
	messages_from_store := make(chan []Message, 1)
	store.Get <- &StoreRequest{zero_time, messages_from_store}
	messages := <-messages_from_store
	if len(messages) != 1 {
		t.FailNow()
	}
	expectMessage(t, &messages[0], at, id, say)
	close(store.Get)
	close(store.Add)
}

func TestFetchBlocksUntilSpeak(t *testing.T) {
	start_fetch_wait_count := fetch_wait_count.String()
	id := "2"
	say := "I've got a lovely fresh cuttle fish for you"
	at := time.Now()
	var zero_time time.Time
	store := start_store()
	messages_from_store := make(chan []Message, 1)
	store.Get <- &StoreRequest{zero_time, messages_from_store}
	for start_fetch_wait_count == fetch_wait_count.String() {
		runtime.Gosched()
	}
	store.Add <- &Message{at, id, say}
	messages := <-messages_from_store
	if len(messages) != 1 {
		t.FailNow()
	}
	expectMessage(t, &messages[0], at, id, say)
	close(store.Get)
	close(store.Add)
}

func TestMultipleListeners(t *testing.T) {
	id := "3"
	say := "This is your nine o'clock alarm call!"
	at := time.Now()
	var zero_time time.Time
	store := start_store()
	const num_clients = 13
	var messages_from_store [num_clients]chan []Message
	for i := 0; i < num_clients; i++ {
		messages_from_store[i] = make(chan []Message, 1)
		store.Get <- &StoreRequest{zero_time, messages_from_store[i]}
	}
	store.Add <- &Message{at, id, say}
	for i := 0; i < num_clients; i++ {
		messages := <-messages_from_store[i]
		if len(messages) != 1 {
			t.FailNow()
		}
		expectMessage(t, &messages[0], at, id, say)
	}
	close(store.Get)
	close(store.Add)
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

func TestPartialRetreive(t *testing.T) {
	start_speak_count := atoi(speak_count.String())
	id1 := "4"
	id2 := "5"
	id3 := "6"
	say1 := "No, no.....No, 'e's stunned!"
	say2 := "You stunned him, just as he was wakin' up!"
	say3 := "Norwegian Blues stun easily, major."
	base := time.Now()
	at1 := base.Add(parseDuration("-4m"))
	since := base.Add(parseDuration("-3m"))
	at2 := base.Add(parseDuration("-2m"))
	at3 := base.Add(parseDuration("-1m"))
	store := start_store()
	store.Add <- &Message{at1, id1, say1}
	store.Add <- &Message{at2, id2, say2}
	store.Add <- &Message{at3, id3, say3}
	for atoi(speak_count.String()) != start_speak_count+3 {
		runtime.Gosched()
	}
	messages_from_store := make(chan []Message, 1)
	store.Get <- &StoreRequest{since, messages_from_store}
	messages := <-messages_from_store
	if len(messages) != 2 {
		t.FailNow()
	}
	expectMessage(t, &messages[0], at2, id2, say2)
	expectMessage(t, &messages[1], at3, id3, say3)
	close(store.Get)
	close(store.Add)
}

func TestPrecisePartialRetreive(t *testing.T) {
	start_speak_count := atoi(speak_count.String())
	id1 := "7"
	id2 := "8"
	id3 := "9"
	say1 := "Well, he's...he's, ah...probably pining for the fjords."
	say2 := "PININ' for the FJORDS?!?!?!?"
	say3 := "look, why did he fall flat on his back the moment I got 'im home?"
	base := time.Now()
	at1 := base.Add(parseDuration("-3m"))
	at2 := base.Add(parseDuration("-2m"))
	at3 := base.Add(parseDuration("-1m"))
	since := at2
	store := start_store()
	store.Add <- &Message{at1, id1, say1}
	store.Add <- &Message{at2, id2, say2}
	store.Add <- &Message{at3, id3, say3}
	for atoi(speak_count.String()) != start_speak_count+3 {
		runtime.Gosched()
	}
	messages_from_store := make(chan []Message, 1)
	store.Get <- &StoreRequest{since, messages_from_store}
	messages := <-messages_from_store
	if len(messages) != 1 {
		t.FailNow()
	}
	expectMessage(t, &messages[0], at3, id3, say3)
	close(store.Get)
	close(store.Add)
}

func TestTypicalFlow(t *testing.T) {
	id1 := "10"
	id2 := "11"
	say1 := "The Norwegian Blue prefers kippin' on it's back!"
	say2 := "Remarkable bird, innit, squire?  Lovely plumage!"
	store := start_store()

	// A waiting zero-time fetch.
	var zero_time time.Time
	prev_fetch_wait_count := fetch_wait_count.String()
	fetch1 := make(chan []Message, 1)
	store.Get <- &StoreRequest{zero_time, fetch1}
	for prev_fetch_wait_count == fetch_wait_count.String() {
		runtime.Gosched()
	}

	// Someone speaks.  This triggers delivery.
	at1 := time.Now()
	store.Add <- &Message{at1, id1, say1}
	messages1 := <-fetch1
	if len(messages1) != 1 {
		t.FailNow()
	}
	expectMessage(t, &messages1[0], at1, id1, say1)

	// Upon recipt, client blocks on fetch with since=at1
	prev_fetch_wait_count = fetch_wait_count.String()
	fetch2 := make(chan []Message, 1)
	store.Get <- &StoreRequest{at1, fetch2}
	for prev_fetch_wait_count == fetch_wait_count.String() {
		runtime.Gosched()
	}

	// Someone speaks again.  This triggers another delivery.
	at2 := time.Now()
	if !at2.After(at1) {
		t.Fail()
	}
	store.Add <- &Message{at2, id2, say2}
	messages2 := <-fetch2
	if len(messages2) != 1 {
		t.FailNow()
	}
	expectMessage(t, &messages2[0], at2, id2, say2)

	close(store.Get)
	close(store.Add)
}
