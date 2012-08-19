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

import "container/list"
import "encoding/json"
import "expvar"
import "flag"
import "log"
import "net/http"
import "strconv"
import "time"

var port = flag.Int("port", 21059, "Port to listen on")
var localaddress = flag.String("localaddress", "", "Local address to bind to")
var max_messages = flag.Int("maxmessages", 1000, "Maximum number of messages to retain")

var start_time = expvar.NewInt("start_time")
var speak_count = expvar.NewInt("speak_count")
var fetch_count = expvar.NewInt("fetch_count")
var fetch_wait_count = expvar.NewInt("fetch_wait_count")
var fetch_wake_count = expvar.NewInt("fetch_wake_count")
var drop_due_to_limit_count = expvar.NewInt("drop_due_to_limit_count")

type Message struct {
	Time time.Time
	ID   string
	Text string
}

type StoreRequest struct {
	StartTime time.Time
	Messages  chan<- []Message
}

type Store struct {
	Add chan *Message
	Get chan *StoreRequest
}

var monotonic_clock chan chan time.Time

const minimum_clock_increment = time.Millisecond

func start_clock() {
	internal_monotonic_clock := make(chan chan time.Time, 1)
	go func() {
		last_time := time.Now()
	main:
		for {
			select {
			case request, ok := <-internal_monotonic_clock:
				if !ok {
					break main
				}
				earliest_acceptable_time := last_time.Add(minimum_clock_increment)
				current_time := time.Now()
				if current_time.Before(earliest_acceptable_time) {
					current_time = earliest_acceptable_time
				}
				request <- current_time
				last_time = current_time
			}
		}
	}()
	monotonic_clock = internal_monotonic_clock
}

func now() time.Time {
	c := make(chan time.Time, 0)
	monotonic_clock <- c
	return <-c
}

func manage_store(store Store) {
	messages := list.New()
	message_count := 0
	waiting := list.New()
main:
	for {
		select {
		case new_message, ok := <-store.Add:
			if !ok {
				break main
			}
			speak_count.Add(1)
			for waiter := waiting.Front(); waiter != nil; waiter = waiter.Next() {
				waiter.Value.(*StoreRequest).Messages <- []Message{*new_message}
				close(waiter.Value.(*StoreRequest).Messages)
				fetch_wake_count.Add(1)
			}
			waiting.Init()
			messages.PushBack(new_message)
			if message_count < *max_messages {
				message_count++
			} else {
				messages.Remove(messages.Front())
				drop_due_to_limit_count.Add(1)
			}
		case request, ok := <-store.Get:
			if !ok {
				break main
			}
			fetch_count.Add(1)
			if messages.Back() == nil || !request.StartTime.Before(messages.Back().Value.(*Message).Time) {
				waiting.PushBack(request)
				fetch_wait_count.Add(1)
			} else {
				start := messages.Back()
				response_size := 1
				if messages.Front().Value.(*Message).Time.After(request.StartTime) {
					start = messages.Front()
					response_size = message_count
				} else {
					for start.Prev().Value.(*Message).Time.After(request.StartTime) {
						start = start.Prev()
						response_size++
					}
				}
				response_messages := make([]Message, 0, response_size)
				for m := start; m != nil; m = m.Next() {
					response_messages = append(response_messages, *m.Value.(*Message))
				}
				request.Messages <- response_messages
			}
		}
	}
}

func start_store() Store {
	store := Store{make(chan *Message, 20), make(chan *StoreRequest, 20)}
	go manage_store(store)
	return store
}

const robots_txt = `User-agent: *
Disallow: /
`

func start_server(store Store) {
	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		var since time.Time
		url_since := r.FormValue("since")
		if url_since != "" {
			err := json.Unmarshal([]byte(url_since), &since)
			if err != nil {
				log.Print("fetch: parse since: ", err)
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Could not parse since as date"))
				return
			}
		}
		messages_from_store := make(chan []Message, 1)
		store.Get <- &StoreRequest{since, messages_from_store}

		json_encoded, err := json.Marshal(<-messages_from_store)
		if err != nil {
			log.Print("json encode: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Write(json_encoded)
	})

	http.HandleFunc("/speak", func(w http.ResponseWriter, r *http.Request) {
		store.Add <- &Message{
			now(),
			r.FormValue("id"),
			r.FormValue("text")}
		w.Header().Add("Access-Control-Allow-Origin", "*")
	})

	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(robots_txt))
	})

	log.Fatal(http.ListenAndServe(*localaddress+":"+strconv.Itoa(*port), nil))
}

func main() {
	flag.Parse()
	start_clock()
	start_time.Set(now().UnixNano())
	store := start_store()
	start_server(store)
}
