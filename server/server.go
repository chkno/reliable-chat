package main

import "container/list"
import "encoding/json"
import "flag"
import "log"
import "net/http"
import "strconv"
import "time"

var port = flag.Int("port", 21059, "Port to listen on")

type Message struct {
	Time time.Time
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

// TODO: Monotonic clock

func manage_store(store Store) {
	messages := list.New()
	message_count := 0
	max_messages := 1000
	waiting := list.New()
main:
	for {
		select {
		case new_message, ok := <-store.Add:
			if !ok {
				break main
			}
			messages.PushBack(new_message)
			for waiter := waiting.Front(); waiter != nil; waiter = waiter.Next() {
				waiter.Value.(*StoreRequest).Messages <- []Message{*new_message}
				close(waiter.Value.(*StoreRequest).Messages)
			}
			waiting.Init()
			if message_count < max_messages {
				message_count++
			} else {
				messages.Remove(messages.Front())
			}
		case request, ok := <-store.Get:
			if !ok {
				break main
			}
			if messages.Back() == nil || !request.StartTime.Before(messages.Back().Value.(*Message).Time) {
				waiting.PushBack(request)
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
		w.Write(json_encoded)
	})

	http.HandleFunc("/speak", func(w http.ResponseWriter, r *http.Request) {
		store.Add <- &Message{time.Now(), r.FormValue("text")}
	})

	log.Fatal(http.ListenAndServe(":" + strconv.Itoa(*port), nil))
}

func main() {
	store := start_store()
	start_server(store)
}
