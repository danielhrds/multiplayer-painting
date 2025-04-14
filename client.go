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
var clientLogger = log.New(os.Stdout, "[CLIENT]: ", log.LstdFlags)
var clientEventsToSend = make(chan *Event)

func _client() error {
	// conn, err := net.Dial("tcp", "26.57.33.158:3120")
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

	// for {
	// 	if !drawing {
	// 		continue
	// 	}
		
		//pixel := <-pixels_ch
		//last_pos := <-last_pos_ch
		//if pixel.Center != last_pos {
		//	bin_buf, err := Encode(pixel)
		//	if err != nil {
		//		log.Fatal(err)
		//	}

		//	_, err = conn.Write(bin_buf.Bytes())
		//	if err != nil {
		//		continue
		//	}
		//}
		// time.Sleep(time.Second / 144)
	// }
	return nil
}

func ClientRead(conn net.Conn) {
	defer conn.Close()
	// buf := make([]byte, 512)
	for {
		// _, err := conn.Read(buf)
		// if err != nil {
		// 	clientLogger.Println("Error trying to read from conn")
		// 	clientLogger.Println(err)
		// 	continue
		// }

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
		last := len(players[event.PlayerId].Scribbles)-1
		if last >= 0 {
			players[event.PlayerId].Scribbles[last] = append(players[event.PlayerId].Scribbles[last], innerEvent.Pixel)
		}
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
		// last := len(players[event.PlayerId].Scribbles)-1
		// players[event.PlayerId].Scribbles[last] = append(players[event.PlayerId].Scribbles[last], innerEvent.Pixel)
	default:
		clientLogger.Println("SENDING: Unknown event type")
	}
}

func StartClient() {	
	wg.Add(1)
	_client()
}
