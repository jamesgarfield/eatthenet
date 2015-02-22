package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jamesgarfield/bingo"
	"github.com/jamesgarfield/hipchat-go/hipchat"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/fcgi"
	"runtime"
	"time"
)

//Command line flags. No flags defaults to fastCGI over standard IO
var (
	local = flag.String("local", "", "serve as webserver, example: 0.0.0.0:8000")
	tcp   = flag.String("tcp", "", "serve as FCGI via TCP, example: 0.0.0.0:8000")
	unix  = flag.String("unix", "", "serve as FCGI via UNIX socket, example: /tmp/myprogram.sock")
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().UnixNano())
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/adblocks/hipchat/message", adbRoomMessage)

	flag.Parse()
	var err error

	if *local != "" { // Run as a local web server
		err = http.ListenAndServe(*local, r)
	} else if *tcp != "" { // Run as FCGI via TCP
		listener, err := net.Listen("tcp", *tcp)
		if err != nil {
			log.Fatal(err)
		}
		defer listener.Close()

		err = fcgi.Serve(listener, r)
	} else if *unix != "" { // Run as FCGI via UNIX socket
		listener, err := net.Listen("unix", *unix)
		if err != nil {
			log.Fatal(err)
		}
		defer listener.Close()

		err = fcgi.Serve(listener, r)
	} else { // Run as FCGI via standard I/O
		err = fcgi.Serve(nil, r)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func adbRoomMessage(w http.ResponseWriter, r *http.Request) {

	handleError := func(err error) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println("Error: ", err.Error())
	}

	data, err := ioutil.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		handleError(err)
		return
	}

	message := &hipchat.RoomMessage{}
	err = message.UnmarshallJSON(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	imgReq := bingo.ImageRequest{message.Item.Message.Message, "Moderate", BING_ACCOUNT_KEY}

	results, err := imgReq.Do()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	img := results[rand.Intn(len(results))]

	notification := &hipchat.NotificationRequest{
		Message:       img.MediaUrl,
		MessageFormat: "text",
	}
	hcClient := hipchat.NewClient(HIPCHAT_TOKEN)
	_, err = hcClient.Room.Notification(message.Item.Room.Id, notification)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
}

func listRooms() {
	hcClient := hipchat.NewClient(HIPCHAT_TOKEN)
	list, _, err := hcClient.Room.List()

	if err != nil {
		panic(err)
	}

	for _, room := range list.Items {
		fmt.Printf("Name: %s  ID: %d\n", room.Name, room.ID)
	}
}
