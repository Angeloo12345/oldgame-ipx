package main

import (
	"bytes"
	"encoding/binary"
	"sync"

	"github.com/gorilla/websocket"
)

type IpxHandler struct {
	rooms         sync.Map
	serverAddress string
}

type IpxRoom struct {
	clients *sync.Map
}

func (handler *IpxHandler) OnConnect(conn *websocket.Conn, room string) {
	address := conn.RemoteAddr().String()
	ipxRoom, _ := handler.rooms.LoadOrStore(room, &IpxRoom{
		clients: &sync.Map{},
	})
	clients := ipxRoom.(*IpxRoom).clients
	prev, loaded := clients.Swap(address, conn)
	if loaded {
		prev.(*websocket.Conn).Close()
	}
}

func (handler *IpxHandler) OnMessage(conn *websocket.Conn, room string, data []byte) {
	header := IPXHeader{}
	header.fromBytes(data)

	if header.Dest.Socket == 0x2 && header.Dest.Host == 0x0 {
		// registration
		header.CheckSum = 0xffff
		header.Length = 30
		header.TransControl = 0
		header.PType = 0

		header.Dest.Network = 0
		header.Dest.setAddress(conn.RemoteAddr().String())
		header.Dest.Socket = 0x2

		header.Src.Network = 1
		header.Src.setAddress(handler.serverAddress)
		header.Src.Socket = 0x2

		conn.WriteMessage(websocket.BinaryMessage, header.toBytes())
	} else {
		ipxRoom, ok := handler.rooms.Load(room)
		if !ok {
			return
		}
		clients := ipxRoom.(*IpxRoom).clients

		if header.Dest.Host == 0xffffffff {
			// broadcast
			clients.Range(func(address, dest interface{}) bool {
				if conn != dest {
					dest.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, data)
				}
				return true
			})
		} else {
			dest, ok := clients.Load(header.Dest.Address())
			if ok {
				dest.(*websocket.Conn).WriteMessage(websocket.BinaryMessage, data)
			}
		}
	}
}

func (handler *IpxHandler) OnClose(conn *websocket.Conn, room string) {
	address := conn.RemoteAddr().String()
	ipxRoom, ok := handler.rooms.Load(room)
	if ok {
		clients := ipxRoom.(*IpxRoom).clients
		clients.Delete(address)
		empty := true
		clients.Range(func(_, _ interface{}) bool {
			empty = false
			return false
		})
		// @caiiiycuk: we can accidentialy delete non empty room,
		if empty {
			handler.rooms.Delete(room)
		}
	}
}

type IPXTransport struct {
	Network uint32
	Host    uint32
	Port    uint16
	Socket  uint16
}

type IPXHeader struct {
	CheckSum     uint16
	Length       uint16
	TransControl uint8
	PType        uint8

	Dest IPXTransport
	Src  IPXTransport
}

func (header *IPXHeader) toBytes() []byte {
	var payload bytes.Buffer
	binary.Write(&payload, binary.BigEndian, header.CheckSum)
	binary.Write(&payload, binary.BigEndian, header.Length)
	binary.Write(&payload, binary.BigEndian, header.TransControl)
	binary.Write(&payload, binary.BigEndian, header.PType)

	binary.Write(&payload, binary.BigEndian, header.Dest.Network)
	binary.Write(&payload, binary.BigEndian, header.Dest.Host)
	binary.Write(&payload, binary.BigEndian, header.Dest.Port)
	binary.Write(&payload, binary.BigEndian, header.Dest.Socket)

	binary.Write(&payload, binary.BigEndian, header.Src.Network)
	binary.Write(&payload, binary.BigEndian, header.Src.Host)
	binary.Write(&payload, binary.BigEndian, header.Src.Port)
	binary.Write(&payload, binary.BigEndian, header.Src.Socket)
	return payload.Bytes()
}

func (header *IPXHeader) fromBytes(data []byte) {
	payload := bytes.NewReader(data)
	binary.Read(payload, binary.BigEndian, &header.CheckSum)
	binary.Read(payload, binary.BigEndian, &header.Length)
	binary.Read(payload, binary.BigEndian, &header.TransControl)
	binary.Read(payload, binary.BigEndian, &header.PType)

	binary.Read(payload, binary.BigEndian, &header.Dest.Network)
	binary.Read(payload, binary.BigEndian, &header.Dest.Host)
	binary.Read(payload, binary.BigEndian, &header.Dest.Port)
	binary.Read(payload, binary.BigEndian, &header.Dest.Socket)

	binary.Read(payload, binary.BigEndian, &header.Src.Network)
	binary.Read(payload, binary.BigEndian, &header.Src.Host)
	binary.Read(payload, binary.BigEndian, &header.Src.Port)
	binary.Read(payload, binary.BigEndian, &header.Src.Socket)
}
