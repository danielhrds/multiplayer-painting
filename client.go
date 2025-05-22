package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

func (b *Board) StartClient() error {
	b.Wg.Add(1)

	url := fmt.Sprintf("localhost:%d", port)
	conn, err := net.Dial("tcp", url)
	if err != nil {
		return err
	}

	encondedEvent, _ := Encode(Event{
		PlayerId:   b.Me.Id,
		Kind:       "ping",
		InnerEvent: PingEvent{},
	})

	length := int32(len(encondedEvent.Bytes()))
	if err := binary.Write(conn, binary.BigEndian, length); err != nil {
		b.Client.Logger.Println("Failed to read prefix length")
		panic(err)
	}

	conn.Write(encondedEvent.Bytes())

	go b.Client.ClientRead(b, conn)
	go b.Client.CSendEvent(b, conn)

	return nil
}

func (bc *BoardClient) ClientRead(board *Board, conn net.Conn) {
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
			bc.Logger.Println("Failed decoding event", err)
			panic(err)
		}

		if event != nil {
			bc.CHandleReceivedEvents(board, event, conn)
		}
	}
}

func (bc *BoardClient) CHandleReceivedEvents(board *Board, event *Event, conn net.Conn) {
	switch innerEvent := event.InnerEvent.(type) {
	case PongEvent:
		board.Client.Logger.Println("Player ID received", event.PlayerId)
		board.Me.Id = event.PlayerId
		bc.EnqueueEvent(board.Me.Id, "joined", JoinedEvent{})
	case JoinedEvent:
		board.Client.Logger.Println("Player joined", innerEvent.Id)
		// avoid recreating the board.Me PlayerObject
		if innerEvent.Id == board.Me.Id {
			bc.AddPlayer(board.Me)
			break
		}
		bc.AddPlayer(NewPlayer(innerEvent.Id))
		bc.Players[innerEvent.Id].Drawing = innerEvent.Drawing
		// bc.Players[innerEvent.Id].Scribbles = innerEvent.Scribbles
		bc.Players[innerEvent.Id].JustJoined = true

		for _, scribble := range innerEvent.Scribbles {
			s := NewScribble(scribble)
			Append(&bc.Players[innerEvent.Id].Scribbles, s)
			cache := bc.NewCache()
			Append(&bc.CacheArray, cache)
			bc.Players[innerEvent.Id].CachedScribbles = append(bc.Players[innerEvent.Id].CachedScribbles, cache)
		}

		board.Changed = true
	case LeftEvent:
		board.Client.Logger.Println("Player left", event.PlayerId)
		// delete(players, event.PlayerId)
	case StartedEvent:
		board.Client.Logger.Println("Player started drawing", event.PlayerId)
		bc.Players[event.PlayerId].Drawing = true
		newScribble := NewScribble([]*Pixel{})
		newScribble.BoundingBox = NewBoundingBox()
		Append(&bc.Players[event.PlayerId].Scribbles, newScribble)

		cache := bc.NewCache()
		Append(&bc.CacheArray, cache)
		Append(&bc.Players[event.PlayerId].CachedScribbles, cache)
	case DoneEvent:
		board.Client.Logger.Println("Player done drawing", event.PlayerId)
		bc.Players[event.PlayerId].Drawing = false
		bc.Players[event.PlayerId].CachedScribbles[len(bc.Players[event.PlayerId].CachedScribbles)-1].Drawing = false
		board.Changed = false
	case DrawingEvent:
		board.Client.Logger.Println("Player sending pixels", event.PlayerId)
		maxIndex := len(bc.Players[event.PlayerId].Scribbles) - 1
		if maxIndex >= 0 {
			scribble := &bc.Players[event.PlayerId].Scribbles[maxIndex]
			var min, max = GetMinAndMax(scribble.BoundingBox.Min, scribble.BoundingBox.Max, innerEvent.Pixel)
			scribble.BoundingBox.Min = min
			scribble.BoundingBox.Max = max
			pixels := &scribble.Pixels
			Append(pixels, innerEvent.Pixel)
		}
		board.Changed = true
	case UndoEvent:
		maxIndex := len(bc.Players[event.PlayerId].Scribbles) - 1
		if maxIndex >= 0 {
			bc.Players[event.PlayerId].Scribbles = bc.Players[event.PlayerId].Scribbles[:maxIndex]
		}

		maxIndex = len(bc.Players[event.PlayerId].CachedScribbles) - 1
		if maxIndex >= 0 {
			bc.Players[event.PlayerId].CachedScribbles = bc.Players[event.PlayerId].CachedScribbles[:maxIndex]
			bc.CacheArray = bc.CacheArray[:len(bc.CacheArray)-1]
		}

		board.SelectedBoundingBox = nil
		board.Changed = true
	case RedoEvent:
		Append(&bc.Players[event.PlayerId].Scribbles, NewScribble(innerEvent.Pixels))

		cache := bc.NewCache()
		Append(&bc.CacheArray, cache)
		Append(&bc.Players[event.PlayerId].CachedScribbles, cache)

		board.Changed = true
		bc.Players[event.PlayerId].Drawing = true
	default:
		board.Client.Logger.Println("Unknown event type")
	}
}

func (bc *BoardClient) CSendEvent(board *Board, conn net.Conn) {
	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()

	var batchedEvents []*Event
	for {
		select {
		case event := <-bc.EventsToSend:
			Append(&batchedEvents, event)

			if len(batchedEvents) > 50 {
				for _, event := range batchedEvents {
					bc.HandleEvent(board, event, conn)
				}
				batchedEvents = batchedEvents[:0]
			}
		case <-ticker.C:
			if len(batchedEvents) > 0 {
				for _, event := range batchedEvents {
					bc.HandleEvent(board, event, conn)
				}
				batchedEvents = batchedEvents[:0]
			}
		}
	}
}

func (bc *BoardClient) HandleEvent(board *Board, event *Event, conn net.Conn) {
	encondedEvent, err := Encode(*event)
	if err != nil {
		bc.Logger.Println("Failed to encode event")
		panic(err)
	}
	length := int32(len(encondedEvent.Bytes()))
	if err := binary.Write(conn, binary.BigEndian, length); err != nil {
		bc.Logger.Println("Failed to write prefix length")
		panic(err)
	}
	switch event.InnerEvent.(type) {
	case JoinedEvent:
		bc.Logger.Println("SENDING: Player joined", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case LeftEvent:
		bc.Logger.Println("SENDING: Player left", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
		board.Wg.Done()
	case StartedEvent:
		bc.Logger.Println("SENDING: Player started drawing", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case DoneEvent:
		bc.Logger.Println("SENDING: Player done drawing", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case DrawingEvent:
		bc.Logger.Println("SENDING: Player sending pixels", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case RedoEvent:
		bc.Logger.Println("SENDING: Player sending redo", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	case UndoEvent:
		bc.Logger.Println("SENDING: Player sending undo", event.PlayerId)
		conn.Write(encondedEvent.Bytes())
	default:
		bc.Logger.Println("SENDING: Unknown event type")
	}
}
