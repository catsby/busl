package busl

import (
	"net/http"
	"bufio"
	"github.com/cyberdelia/pat"
	"log"
	"io"
)

func mkstream(w http.ResponseWriter, r *http.Request) {
	uuid, err := uuid()
	if err != nil {
		log.Printf("%v", err)
		http.Error(w, "Unable to create stream. Please try again.", http.StatusServiceUnavailable)
		return
	}
	io.WriteString(w, uuid)
}

func pub(w http.ResponseWriter, r *http.Request) {
	uuid := UUID(r.URL.Query().Get(":uuid"))

	msgBroker := NewRedisBroker(uuid)
	defer msgBroker.UnsubscribeAll()

	scanner := bufio.NewScanner(r.Body)

	for scanner.Scan() {
		msgBroker.Publish(scanner.Bytes())
	}
}

func sub(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	uuid := UUID(r.URL.Query().Get(":uuid"))

	msgBroker := NewRedisBroker(uuid)
	ch := msgBroker.Subscribe()
	defer msgBroker.UnsubscribeAll()

	for msg := range ch {
		w.Write(msg)
		f.Flush()
	}
}

func Start() {
	p := pat.New()

	p.PostFunc("/streams", mkstream)
	p.PostFunc("/streams/:uuid", pub)
	p.GetFunc("/streams/:uuid", sub)

	http.Handle("/", p)

	if err := http.ListenAndServe(":" + *httpPort, nil); err != nil {
		panic(err)
	}
}
