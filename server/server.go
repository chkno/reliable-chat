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

var frame_count = expvar.NewInt("frame_count")
var speak_count = expvar.NewInt("speak_count")
var fetch_count = expvar.NewInt("fetch_count")
var fetch_wait_count = expvar.NewInt("fetch_wait_count")
var fetch_wake_count = expvar.NewInt("fetch_wake_count")

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
			speak_count.Add(1)
			for waiter := waiting.Front(); waiter != nil; waiter = waiter.Next() {
				waiter.Value.(*StoreRequest).Messages <- []Message{*new_message}
				close(waiter.Value.(*StoreRequest).Messages)
				fetch_wake_count.Add(1)
			}
			waiting.Init()
			messages.PushBack(new_message)
			if message_count < max_messages {
				message_count++
			} else {
				messages.Remove(messages.Front())
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

const frame_html = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN"
  "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">

<html xmlns="http://www.w3.org/1999/xhtml">
<head>
 <script type="text/javascript"><!--//--><![CDATA[//><!--
  var since;
  function go() {
   var delay = 10000;
   var xhr = new XMLHttpRequest();
   xhr.onreadystatechange = function() {
    if (this.readyState == this.DONE) {
     if (this.status == 200) {
      var rtxt = this.responseText;
      if (rtxt != null) {
       var r = JSON.parse(rtxt);
       if (r != null) {
        window.parent.postMessage(rtxt, "*");
        delay = 40;
        if (r.length >= 1 && "Time" in r[r.length-1]) {
         since = r[r.length-1]["Time"];
        }
       }
      }
     }
     window.setTimeout(go, delay);
    }
   }
   var uri = "/fetch";
   if (since) {
    uri += '?since="' + since + '"';
   }
   xhr.open("GET", uri);
   xhr.send();
  }
  //--><!]]></script>
</head>
<body onload="go()">
</body>
</html>
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
			time.Now(),
			r.FormValue("id"),
			r.FormValue("text")}
	})

	http.HandleFunc("/frame", func(w http.ResponseWriter, r *http.Request) {
		frame_count.Add(1)
		w.Write([]byte(frame_html));
	})

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

func main() {
	flag.Parse()
	store := start_store()
	start_server(store)
}
