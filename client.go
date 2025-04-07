package main

import (
	"log"
	"net"
	"time"
)

func _client() error {
	// conn, err := net.Dial("tcp", "26.57.33.158:3120")
	conn, err := net.Dial("tcp", "localhost:3120")
	if err != nil {
		return err
		// log.Fatal(err)
	}

	go ClientRead(conn)

	for {
		if !drawing {
			continue
		}

		pixel := <-pixels_ch
		last_pos := <-last_pos_ch
		if pixel.Center != last_pos {
			bin_buf, err := Encode(pixel)
			if err != nil {
				log.Fatal(err)
			}

			_, err = conn.Write(bin_buf.Bytes())
			if err != nil {
				continue
			}
		}
		time.Sleep(time.Second / 144)
	}
}

func ClientRead(conn net.Conn) {
	buf := make([]byte, 156)
	for {
		_, err := conn.Read(buf)
		if err != nil {
			log.Println(err)
			continue
		}

		data, err := Decode(buf)
		if err != nil {
			log.Println(err)
		}

		if data != nil {
			// fmt.Println("received data from SERVER: ")
			// fmt.Println("data: ", data)

			buffer_test_paint = append(buffer_test_paint, data)
			changed = true
		}

	}
}

func StartClient() {
	_client()
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
}
