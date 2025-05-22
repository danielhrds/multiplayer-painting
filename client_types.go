package main

import (
	"log"
	"os"
	"sync"
	"sync/atomic"
	
	rl "github.com/gen2brain/raylib-go/raylib"
)

type BoardClient struct {
	Players         map[int32]*Player
	Me              *Player
	Logger          *Logger
	EventsToSend    chan *Event
	CacheArray      []*Cache // This exists because golang maps are unordered
	CacheLayerIndex int32
}

func NewBoardClient() *BoardClient {
	return &BoardClient{
		Players:         make(map[int32]*Player),
		Me:              NewPlayer(0),
		Logger:          NewLogger(os.Stdout, "[CLIENT]: ", log.LstdFlags),
		EventsToSend:    make(chan *Event),
		CacheArray:      []*Cache{},
		CacheLayerIndex: 0,
	}
}

func (bc *BoardClient) EnqueueEvent(playerId int32, kind string, innerEvent any) {
	bc.EventsToSend <- &Event{
		PlayerId:   playerId,
		Kind:       kind,
		InnerEvent: innerEvent,
	}
}

type Board struct {
	Width               int32
	Height              int32
	LastMousePos        rl.Vector2
	Changed             bool
	PixelSize           float32
	FPS                 int32
	FrameCount          int32
	FrameSpeed          int32
	Wg                  sync.WaitGroup
	UiMode              bool
	SelectedColor       rl.Color
	CONFIG_COLOR        rl.Color
	ColorPicker         ColorPicker
	ColorPickerOpened   bool
	SelectedBoundingBox *BoundingBox
	Me                  *Player
	Client              *BoardClient
}

func NewBoard() *Board {
	return &Board{
		Width:         1600,
		Height:        900,
		LastMousePos:  rl.Vector2{},
		Changed:       false,
		PixelSize:     10,
		FPS:           60,
		FrameCount:    0,
		FrameSpeed:    30,
		Wg:            sync.WaitGroup{},
		UiMode:        true,
		SelectedColor: rl.Black,
		CONFIG_COLOR:  rl.Magenta,
		ColorPicker: ColorPicker{
			Colors: []rl.Color{
				rl.Black,
				rl.Blue,
				rl.Pink,
				rl.Purple,
				rl.Yellow,
				rl.Orange,
				rl.Red,
				rl.Green,
			},
			Center: rl.Vector2{},
			Radius: 120,
		},
		ColorPickerOpened:   false,
		SelectedBoundingBox: nil,
		Me:                  NewPlayer(0),
		Client:         		 NewBoardClient(),
	}
}

type Cache struct {
	Drawing, Empty  bool
	RenderTexture2D *rl.RenderTexture2D
	LayerIndex      int32
}

func (bc *BoardClient) NewCache() *Cache {
	newLayerIndex := atomic.AddInt32(&bc.CacheLayerIndex, 1)
	return &Cache{
		Drawing:         true,
		Empty:           true,
		RenderTexture2D: &rl.RenderTexture2D{},
		LayerIndex:      newLayerIndex,
	}
}

type Player struct {
	Id              int32
	Drawing         bool
	JustJoined      bool
	Scribbles       []Scribble
	CachedScribbles []*Cache
}

func NewPlayer(id int32) *Player {
	return &Player{
		id,
		false,
		false,
		make([]Scribble, 0),
		make([]*Cache, 0),
	}
}

