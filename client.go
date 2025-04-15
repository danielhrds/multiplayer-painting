package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"time"
)

var players = make(map[int32]*Player)
var me = NewPlayer(0)
var clientLogger = NewLogger(os.Stdout, "[CLIENT]: ", log.LstdFlags)
var clientEventsToSend = make(chan *Event)

func _client() error {
	conn, err := net.Dial("tcp", "localhost:3120")
	if err != nil {
		return err
	}

	encondedEvent, _:= Encode(Event{
		PlayerId: me.Id,
		Kind: "ping",
		InnerEvent: PingEvent{},
	})
	
	length := int32(len(encondedEvent.Bytes()))
	if err := binary.Write(conn, binary.BigEndian, length); err != nil {
		clientLogger.Println("Failed to read prefix length")
		panic(err)
	}
	
	conn.Write(encondedEvent.Bytes())

	go ClientRead(conn)
	go CSendEvent(conn)

	return nil
}

func ClientRead(conn net.Conn) {
	defer conn.Close()

	for {
		var length int32
		if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
			serverLogger.Println("Failed to read length:", err)
			panic(err)
		}

		buf := make([]byte, length)
		if _, err := io.ReadFull(conn, buf); err != nil {
			serverLogger.Println("Failed to read full message:", err)
			panic(err)
		}

		event, err := Decode(buf)
		if err != nil {
			clientLogger.Println("Failed decoding event", err)
			panic(err)
		}
		
		if event != nil {
			CHandleReceivedEvents(event, conn)
		}
	}
}

type Player struct {
	Id int32
	Drawing bool
	Scribbles [][]*Pixel
}

func NewPlayer(id int32) *Player {
	return &Player{
		id,
		false,
		make([][]*Pixel, 0),
	}
}

func CHandleReceivedEvents(event *Event, conn net.Conn) {
	switch innerEvent := event.InnerEvent.(type) {
	case PongEvent:
		clientLogger.Println("Player ID received", event.PlayerId)
		me.Id = event.PlayerId
		clientEventsToSend <- &Event{
			PlayerId: me.Id,
			Kind: "joined",
			InnerEvent: JoinedEvent{},
		}
	case JoinedEvent:
		clientLogger.Println("Player joined", innerEvent.Id)
		// avoid recreating the me PlayerObject
		if innerEvent.Id == me.Id { 
			players[innerEvent.Id] = me
			break 
		} 
		players[innerEvent.Id] = NewPlayer(innerEvent.Id)
		players[innerEvent.Id].Drawing = innerEvent.Drawing
		players[innerEvent.Id].Scribbles = innerEvent.Scribbles
	case LeftEvent:
		clientLogger.Println("Player left", event.PlayerId)
		// delete(players, event.PlayerId)
	case StartedEvent:
		clientLogger.Println("Player started drawing", event.PlayerId)
		players[event.PlayerId].Drawing = true
		players[event.PlayerId].Scribbles = append(players[event.PlayerId].Scribbles, []*Pixel{})
	case DoneEvent:
		clientLogger.Println("Player done drawing", event.PlayerId)
		players[event.PlayerId].Drawing = false
	case DrawingEvent:
		clientLogger.Println("Player sending pixels", event.PlayerId)
		maxIndex := len(players[event.PlayerId].Scribbles)-1
		if maxIndex >= 0 {
			players[event.PlayerId].Scribbles[maxIndex] = append(players[event.PlayerId].Scribbles[maxIndex], innerEvent.Pixel)
		}
		changed = true
	case UndoEvent:
		maxIndex := len(players[event.PlayerId].Scribbles)-1
		if maxIndex >= 0 {
			players[event.PlayerId].Scribbles = players[event.PlayerId].Scribbles[:maxIndex] 
		}
		changed = true
	case RedoEvent:
		players[event.PlayerId].Scribbles = append(players[event.PlayerId].Scribbles, innerEvent.Pixels)
		changed = true
	default:
    clientLogger.Println("Unknown event type")
	}
}

func CSendEvent(conn net.Conn) {
	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()

	var batchedEvents []*Event
	for {
		select {
		case event := <- clientEventsToSend:
			batchedEvents = append(batchedEvents, event)

			if len(batchedEvents) > 50 {
				for _, event := range batchedEvents {
					HandleEvent(event, conn)
				}
				batchedEvents = batchedEvents[:0]
			}
		case <-ticker.C:
			if len(batchedEvents) > 0 {
				for _, event := range batchedEvents {
					HandleEvent(event, conn)
				}
				batchedEvents = batchedEvents[:0]
			}
		}	
	}
}

func HandleEvent(event *Event, conn net.Conn) {
	encondedEvent, err := Encode(*event)
	if err != nil {
		clientLogger.Println("Failed to encode event")
		panic(err)
	}
	length := int32(len(encondedEvent.Bytes()))
	if err := binary.Write(conn, binary.BigEndian, length); err != nil {
		clientLogger.Println("Failed to write prefix length")
		panic(err)
	}
	switch event.InnerEvent.(type) {
	case JoinedEvent:
		clientLogger.Println("SENDING: Player joined", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case LeftEvent:
		clientLogger.Println("SENDING: Player left", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
		wg.Done()
	case StartedEvent:
		clientLogger.Println("SENDING: Player started drawing", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case DoneEvent:
		clientLogger.Println("SENDING: Player done drawing", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case DrawingEvent:
		clientLogger.Println("SENDING: Player sending pixels", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case RedoEvent:
		clientLogger.Println("SENDING: Player sending redo", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case UndoEvent:
		clientLogger.Println("SENDING: Player sending undo", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	default:
		clientLogger.Println("SENDING: Unknown event type")
	}
}

func StartClient() {	
	wg.Add(1)
	_client()
}
