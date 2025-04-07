package main

import (
	"bytes"
	"encoding/gob"
	"net"
)

type Client struct {
	id   int
	conn net.Conn
	pixels *[]*Pixel
}

func NewClient(id int, conn net.Conn) *Client {
	return &Client{
		id,
		conn,
		new([]*Pixel),
	}
}

func Encode(to_encode Pixel) (*bytes.Buffer, error) {
	bin_buf := new(bytes.Buffer)
	gobobj := gob.NewEncoder(bin_buf)
	err := gobobj.Encode(to_encode)
	return bin_buf, err
}

func Decode(buffer []byte) (*Pixel, error) {
	tmpbuffer := bytes.NewBuffer(buffer)
	gobobj := gob.NewDecoder(tmpbuffer)
	data := new(Pixel) // might change name to data
	err := gobobj.Decode(data)
	return data, err
}
