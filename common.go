package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"image/color"
	"io"
	"log"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var logEnabled bool
var port = 3120

func init() {
	flag.BoolVar(&logEnabled, "log", false, "Enable log")
	flag.Parse()
	clientLogger.enabled = logEnabled
	serverLogger.enabled = logEnabled
	
	// events
	gob.Register(Event{})
	gob.Register(JoinedEvent{})
	gob.Register(LeftEvent{})
	gob.Register(StartedEvent{})
	gob.Register(DrawingEvent{})
	gob.Register(DoneEvent{})
	gob.Register(PingEvent{})
	gob.Register(PongEvent{})
	gob.Register(UndoEvent{})
	gob.Register(RedoEvent{})
	
	// nested types (used inside events)
	gob.Register(Pixel{})
	gob.Register(rl.Vector2{})
	gob.Register(rl.Color{})
	gob.Register(color.RGBA{})
	gob.Register([]*Pixel{})
	gob.Register([][]*Pixel{})
}

type Pixel struct {
	Center rl.Vector2
	Radius float32
	Color  rl.Color
}

type Event struct {
	PlayerId int32
	Kind string
	InnerEvent interface{}
}

type PingEvent struct {}

type PongEvent struct {}

type JoinedEvent struct {
	Id int32
	Drawing bool
	Scribbles [][]*Pixel
} // CHANGE TO HAVE THE DATA OF THE OTHER PLAYER INSIDE IT

type LeftEvent struct {}

type DoneEvent struct {}

type StartedEvent struct {}

type DrawingEvent struct {
	Pixel *Pixel
}

type UndoEvent struct {}

type RedoEvent struct {
	Pixels []*Pixel
}

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

type Logger struct {
	logger *log.Logger
	enabled bool
}

func NewLogger(out io.Writer, prefix string, flag int) *Logger {
	return &Logger{
		logger: log.New(os.Stdout, prefix, log.LstdFlags),
		enabled: false,
	}	
}

func (l *Logger) Println(v ...any) {
	if l.enabled {
		l.logger.Println(v...)
	}
}

func Append[T any](array *[]T, toAppend T) {
	*array = append(*array, toAppend)
}