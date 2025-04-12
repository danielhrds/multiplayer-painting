package main

import (
	"bytes"
	"encoding/gob"
	"image/color"
	"net"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func init() {
	// events
	gob.Register(Event{})
	gob.Register(JoinedEvent{})
	gob.Register(LeftEvent{})
	gob.Register(StartedEvent{})
	gob.Register(DrawingEvent{})
	gob.Register(DoneEvent{})
	gob.Register(PingEvent{})
	gob.Register(PongEvent{})
	
	// nested types (used inside events)
	gob.Register(Pixel{})
	gob.Register(rl.Vector2{})
	gob.Register(rl.Color{})
	gob.Register(color.RGBA{})
	gob.Register([]*Pixel{})
	gob.Register([][]*Pixel{})
}

// type Shape = []*Pixel 
// type Drawings = *[]Shape

type Pixel struct {
	Center rl.Vector2
	Radius float32
	Color  rl.Color
}

type Client struct {
	Id   int32
	Conn net.Conn
	Drawing bool
	Scribbles [][]*Pixel
}

type Event struct {
	PlayerId int32
	Kind string
	InnerEvent interface{}
}

type PingEvent struct {}

type PongEvent struct {}

type JoinedEvent struct {}

type LeftEvent struct {}

type DoneEvent struct {}

type StartedEvent struct {}

type DrawingEvent struct {
	Pixel *Pixel
}

func NewClient(id int32, conn net.Conn) *Client {
	return &Client{
		id,
		conn,
		false,
		make([][]*Pixel, 0),
	}
}

func EncodePixel(to_encode Pixel) (*bytes.Buffer, error) {
	bin_buf := new(bytes.Buffer)
	gobobj := gob.NewEncoder(bin_buf)
	err := gobobj.Encode(to_encode)
	return bin_buf, err
}

func DecodePixel(buffer []byte) (*Pixel, error) {
	tmpbuffer := bytes.NewBuffer(buffer)
	gobobj := gob.NewDecoder(tmpbuffer)
	data := new(Pixel) // might change name to data
	err := gobobj.Decode(data)
	return data, err
}

func EncodeArrayPixel(to_encode []Pixel) (*bytes.Buffer, error) {
	bin_buf := new(bytes.Buffer)
	gobobj := gob.NewEncoder(bin_buf)
	err := gobobj.Encode(to_encode)
	return bin_buf, err
}

func DecodeArrayPixel(buffer []byte) (*[]Pixel, error) {
	tmpbuffer := bytes.NewBuffer(buffer)
	gobobj := gob.NewDecoder(tmpbuffer)
	data := new([]Pixel)
	err := gobobj.Decode(data)
	return data, err
}

// need to create a type of union between Pixel and []Pixel
// func (s *StartedPackage) Encode() (*bytes.Buffer, error) {
// 	bin_buf := new(bytes.Buffer)
// 	gobobj := gob.NewEncoder(bin_buf)
// 	err := gobobj.Encode(s)
// 	return bin_buf, err
// }
// 
// func Decode(buffer []byte) (any, error) { // change any to a interface/union of return values
// 
// 	tmpbuffer := bytes.NewBuffer(buffer)
// 	gobobj := gob.NewDecoder(tmpbuffer)
// 	data := new([]Pixel)
// 	err := gobobj.Decode(data)
// 	return data, err
// }


// encode an event to bytes
func Encode(to_encode Event) (*bytes.Buffer, error) {
	bin_buf := new(bytes.Buffer)
	gobobj := gob.NewEncoder(bin_buf)
	err := gobobj.Encode(to_encode)
	return bin_buf, err
}

// decode bytes to event
func Decode(buffer []byte) (*Event, error) {
	tmpbuffer := bytes.NewBuffer(buffer)
	gobobj := gob.NewDecoder(tmpbuffer)
	var event Event
	err := gobobj.Decode(&event)
	return &event, err
}