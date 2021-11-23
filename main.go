package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)


type CustomHandler struct {
	eventQueue chan string
}

func NewCustomHandler(eventQueue chan string) *CustomHandler {
	return &CustomHandler{eventQueue: eventQueue}
}

func consumer(ctx context.Context, eventQueue chan string, 
	doneChan chan interface{}) {
	wg := &sync.WaitGroup{}

	for {
		select {
		
		case <-ctx.Done():
            
			wg.Wait()
			log.Println("writing to done channel")
			doneChan <- struct{}{}
			log.Println("Done, shutting down the consumer")
			return
		case event := <-eventQueue:
			wg.Add(3)
			go event0(event, wg)
			go event1(event, wg)
			go event2(event, wg)
		}
	}
}

func event0(name string, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("starting event 0 for %s\n", name)
	time.Sleep(5 * time.Second)
	log.Printf("finished event 0 for %s\n", name)
}

func event1(name string, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("starting event 1 for %s\n", name)
	time.Sleep(5 * time.Second)
	log.Printf("finished event 1 for %s\n", name)
}

func event2(name string, wg *sync.WaitGroup) {
	defer wg.Done()
	log.Printf("starting event 2 for %s\n", name)
	time.Sleep(5 * time.Second)
	log.Printf("finished event 2 for %s\n", name)
}

func (h *CustomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	eventName := vars["eventName"]

	h.eventQueue <- eventName

	fmt.Fprintf(w, "event %s started", eventName)
}

func main() {

	eventQueue := make(chan string)

	customHandler := NewCustomHandler(eventQueue)

	ctx, cancelFunc := context.WithCancel(context.Background())

	r := mux.NewRouter()
	r.Handle("/{eventName}", customHandler)

	server := &http.Server{
		Addr: ":8080",
		Handler: r,
	}

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGTERM, syscall.SIGINT) 

	go func() {
		if err := server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				log.Printf("HTTP server closed with: %v\n", err)
			}
			log.Printf("HTTP server shut down")
		}
	}()

	doneChan := make(chan interface{})
	go consumer(ctx, eventQueue, doneChan)

	<-termChan
	log.Println("Signal received. Shutdown process initiated")

	

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	
	cancelFunc()
	
	log.Println("waiting consumer to finish its events")
	<-doneChan
	log.Println("done. returning.")

}
