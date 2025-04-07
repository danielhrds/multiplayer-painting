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

package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

type Server struct{}

var clients = make(map[int]*Client)
var clients_buffer = make(map[int]*[]*Pixel)
var id int = 0
var pixels_server_ch = make(chan []byte)

func (s *Server) Start() {
	// ln, err := net.Listen("tcp", "26.57.33.158:3120")
	ln, err := net.Listen("tcp", "localhost:3120")
	if err != nil {
		return
		// log.Fatal(err)
	}

	go SendPixels()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		clients[id] = NewClient(id, conn)
		go s.ReadConn(id, conn)
		id++
	}
}

func (s *Server) ReadConn(id int, conn net.Conn) {
	buf := make([]byte, 156)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			return

		}

		data, err := Decode(buf)
		if err != nil {
			log.Println(err)
		}

		if data != nil {
			// fmt.Println("received data: ")
			// fmt.Println("data: ", data)

			pixels_server_ch <- buf
			*clients[id].pixels = append(*clients[id].pixels, data)
		}

		// if err != nil {
		// 	fmt.Println(errors.Is(err, os.ErrDeadlineExceeded))
		// 	log.Fatal("read error", err)
		// }
	}
}

func SendPixels() {
	// TODO: Tick to accumulate pixels instead of sending every pixel at once
	go func() {
		for {
			for id, client := range clients {
				fmt.Println(id, client.pixels)
			}
			// time.Sleep(time.Second / 144)
			time.Sleep(time.Second / 60)
		}
	}()

	for pixel := range pixels_server_ch {
		for _, v := range clients {
			v.conn.Write(pixel)
		}
	}
}

// test
// func Client() func(size int) error {
// 	conn, err := net.Dial("tcp", "localhost:3120")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	return func(size int) error {
// 		file := make([]byte, size)
// 		_, err := io.ReadFull(rand.Reader, file)
// 		if err != nil {
// 			return err
// 		}

// 		//
// 		bin_buf := new(bytes.Buffer)
// 		gobobj := gob.NewEncoder(bin_buf)
// 		gobobj.Encode(buffer_to_paint)
// 		fmt.Println("buffer: ", *buffer_to_paint[0])
// 		//

// 		n, err := conn.Write(bin_buf.Bytes())
// 		if err != nil {
// 			return err
// 		}

// 		fmt.Println("written", n)
// 		return nil
// 	}
// }

func StartServer() {
	// go func() {
	// 	size := 1000
	// 	time.Sleep(2 * time.Second)
	// 	client := Client()
	// 	for {
	// 		time.Sleep(2 * time.Second)
	// 		client(size)
	// 		size += 1000
	// 	}
	// }()

	server := &Server{}
	server.Start()

}
