// 1. SEND A MAP OF CLIENTS THAT DREW SOMETHING
// 2. NO CHANNELS NEEDED, JUST A MAP THAT GETS READED AFTER A SERVER TICK
// 3. ACCUMULATE CHANGES WITHIN TICKS ON A MAP
//	  AND APPEND TO CLIENT.PIXELS AFTER

// EVENTS RECEIVED BY SERVER
// - JOINED: Add client to clients map. Send all client's pixels to reconstruct the board.
// - DRAW: Accumulate pixels on a pixel buffer. Free after each tick
// - LEFT: Remove client from clients map

// EVENTS SENT BY SERVER
// DRAW: Sends all accumulated pixels.

// (client) send the pixels from the client,
// (server) store them in a array of array of pixels [][]Pixel
// (server) send the pixels from the server back to the client
// what im thinking:
// 		the interaction between client and server can be only sending pixel for pixel
// 		but when other clients interact between each other, the server needs to send the whole array of pixels
// 		the client then only replace the last array of the other client

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync/atomic"
	"time"
)

type Server struct{}

type Client struct {
	Id        int32
	Conn      net.Conn
	Drawing   bool
	Scribbles [][]*Pixel
	Deleted   [][]*Pixel
}

func NewClient(id int32, conn net.Conn) *Client {
	return &Client{
		id,
		conn,
		false,
		make([][]*Pixel, 0),
		make([][]*Pixel, 0),
	}
}

var id int32 = 0
var clients = make(map[int32]*Client)
var eventsToSend = make(chan *Event)
var bytesReceivedWithinTick = 0

var serverLogger = NewLogger(os.Stdout, "[SERVER]: ", log.LstdFlags)

func (s *Server) Start() {
	url := fmt.Sprintf("localhost:%d", port)
	ln, err := net.Listen("tcp", url)
	if err != nil {
		return
	}

	go SendEvent()
	go Tick()

	serverLogger.Println("Server running or port", port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go s.ReadConn(conn)
	}
}

// create the package and store them on update_buffer
func (s *Server) ReadConn(conn net.Conn) {
	defer conn.Close()
	defer func(conn net.Conn) {
		if r := recover(); r != nil {
			serverLogger.Println("Recovered from panic in ReadConn:", r)
		}
	}(conn)

	for {
		var length int32
		if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
			serverLogger.Println("Failed to read length:", err)
			// panic(err)
			return
		}
		bytesReceivedWithinTick += int(length)

		buf := make([]byte, length)
		if _, err := io.ReadFull(conn, buf); err != nil {
			serverLogger.Println("Failed to read full message:", err)
			// panic(err)
			return
		}

		event, err := Decode(buf)
		if err != nil {
			serverLogger.Println(event)
			serverLogger.Println(err)
			// panic(err)
			return
		}

		if event != nil {
			SHandleReceivedEvents(event, conn)
		}
	}
}

// handle event received
func SHandleReceivedEvents(event *Event, conn net.Conn) {
	// stay aware that when just forwarding the events to be sent it may break things
	switch innerEvent := event.InnerEvent.(type) {
	case PingEvent:
		serverLogger.Println("Receiving: Ping received")
		newId := atomic.AddInt32(&id, 1)
		clients[newId] = NewClient(newId, conn)
		eventsToSend <- &Event{
			PlayerId:   newId,
			Kind:       "pong",
			InnerEvent: PongEvent{},
		}
		// maybe use a lock to add the id
	case JoinedEvent:
		serverLogger.Println("Receiving: Joined")
		for _, client := range clients {
			serverLogger.Println("Client Joined", client.Id)
			eventsToSend <- &Event{
				PlayerId: event.PlayerId,
				Kind:     event.Kind,
				InnerEvent: JoinedEvent{
					Id:        client.Id,
					Drawing:   client.Drawing,
					Scribbles: client.Scribbles,
				},
			}
		}
	case LeftEvent:
		serverLogger.Println("Receiving: Left")
		eventsToSend <- &Event{
			PlayerId:   event.PlayerId,
			Kind:       event.Kind,
			InnerEvent: LeftEvent{},
		}
		// don't delete the player, it's useful to
		// rebuild the board when someone enters
		// delete(clients, event.PlayerId)
	case StartedEvent:
		serverLogger.Println("Receiving: Started Drawing")
		clients[event.PlayerId].Drawing = true
		Append(&clients[event.PlayerId].Scribbles, []*Pixel{})
		eventsToSend <- &Event{
			PlayerId:   event.PlayerId,
			Kind:       event.Kind,
			InnerEvent: StartedEvent{},
		}
	case DoneEvent:
		serverLogger.Println("Receiving: Done")
		clients[event.PlayerId].Drawing = false
		eventsToSend <- &Event{
			PlayerId:   event.PlayerId,
			Kind:       event.Kind,
			InnerEvent: DoneEvent{},
		}
	case DrawingEvent:
		serverLogger.Println("Receiving: Player sending pixels")
		maxIndex := len(clients[event.PlayerId].Scribbles) - 1
		if maxIndex >= 0 {
			Append(&clients[event.PlayerId].Scribbles[maxIndex], innerEvent.Pixel)
		}
		eventsToSend <- event
	case UndoEvent:
		serverLogger.Println("Receiving: Player sending undo")
		maxIndex := len(clients[event.PlayerId].Scribbles) - 1
		if maxIndex >= 0 {
			last := clients[event.PlayerId].Scribbles[maxIndex]
			clients[event.PlayerId].Scribbles = clients[event.PlayerId].Scribbles[:maxIndex]
			Append(&clients[event.PlayerId].Deleted, last)
			eventsToSend <- event
		}
	case RedoEvent:
		serverLogger.Println("Receiving: Player sending redo")
		maxIndex := len(clients[event.PlayerId].Deleted) - 1
		if maxIndex >= 0 {
			last := clients[event.PlayerId].Deleted[maxIndex]
			Append(&clients[event.PlayerId].Scribbles, last)
			clients[event.PlayerId].Deleted = clients[event.PlayerId].Deleted[:maxIndex]
			eventsToSend <- &Event{
				PlayerId: event.PlayerId,
				Kind:     "redo",
				InnerEvent: RedoEvent{
					Pixels: last,
				},
			}
		}
	default:
		serverLogger.Println("Receiving: Unknown event type")
	}
}

func SendEvent() {
	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()

	for {
		<-ticker.C
		var events []*Event

	AccumulateEvents:
		for {
			select {
			case event := <-eventsToSend:
				Append(&events, event)
			default:
				break AccumulateEvents
			}
		}

		for _, event := range events {
			encondedEvent, _ := Encode(*event)
			length := int32(len(encondedEvent.Bytes()))
			switch event.InnerEvent.(type) {
			case PongEvent:
				serverLogger.Println("Sending: ID back (PongEvent)", event.PlayerId)
				conn := clients[event.PlayerId].Conn
				if err := binary.Write(conn, binary.BigEndian, length); err != nil {
					return
				}
				conn.Write(encondedEvent.Bytes())
			case JoinedEvent:
				serverLogger.Println("Sending: JoinedEvent", event.PlayerId)
				for _, client := range clients {
					conn := client.Conn
					if err := binary.Write(conn, binary.BigEndian, length); err != nil {
						continue
					}
					conn.Write(encondedEvent.Bytes())
				}
			case LeftEvent:
				serverLogger.Println("Sending: Left", event.PlayerId)
				for _, client := range clients {
					conn := client.Conn
					if err := binary.Write(conn, binary.BigEndian, length); err != nil {
						continue
					}
					conn.Write(encondedEvent.Bytes())
				}
			case StartedEvent:
				serverLogger.Println("Sending: StartedEvent", event.PlayerId)
				for _, client := range clients {
					conn := client.Conn
					if err := binary.Write(conn, binary.BigEndian, length); err != nil {
						continue
					}
					conn.Write(encondedEvent.Bytes())
				}
			case DoneEvent:
				serverLogger.Println("Sending: DoneEvent", event.PlayerId)
				for _, client := range clients {
					conn := client.Conn
					if err := binary.Write(conn, binary.BigEndian, length); err != nil {
						continue
					}
					conn.Write(encondedEvent.Bytes())
				}
			case DrawingEvent:
				serverLogger.Println("Sending: DrawingEvent", event.PlayerId)
				for _, client := range clients {
					conn := client.Conn
					if err := binary.Write(conn, binary.BigEndian, length); err != nil {
						continue
					}
					conn.Write(encondedEvent.Bytes())
				}
			case UndoEvent:
				serverLogger.Println("Sending: UndoEvent", event.PlayerId)
				for _, client := range clients {
					conn := client.Conn
					if err := binary.Write(conn, binary.BigEndian, length); err != nil {
						continue
					}
					conn.Write(encondedEvent.Bytes())
				}
			case RedoEvent:
				serverLogger.Println("Sending: RedoEvent", event.PlayerId)
				for _, client := range clients {
					conn := client.Conn
					if err := binary.Write(conn, binary.BigEndian, length); err != nil {
						continue
					}
					conn.Write(encondedEvent.Bytes())
				}
			default:
				serverLogger.Println("Sending: Unknown event type")
			}
		}
	}
}

func Tick() {
	ticker := time.NewTicker(time.Second / 60)

	counter := 0

	for {
		<-ticker.C
		if counter%60 == 0 {
			serverLogger.Println("MB: ", prettySIByteSize(bytesReceivedWithinTick))
		}
		counter++
	}
}

func prettySIByteSize(b int) string {
	bf := float64(b)
	for _, unit := range []string{"", "k", "M", "G", "T", "P", "E", "Z"} {
		if bf < 1000.0 {
			return fmt.Sprintf("%.1f %sB", bf, unit)
		}
		bf /= 1000.0
	}
	return fmt.Sprintf("%.1f YB", bf)
}

func StartServer() {
	server := &Server{}
	server.Start()
}
