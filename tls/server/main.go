package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
)

type Room struct {
	mutx  sync.Mutex
	users map[string]net.Conn
	queue uint
}

func (r *Room) New() *Room {
	r.mutx = sync.Mutex{}
	r.users = map[string]net.Conn{}
	return r
}

func (r *Room) AddUser(conn net.Conn, id string) {
	r.mutx.Lock()
	defer r.mutx.Unlock()
	r.users[id] = conn
}

func (r *Room) UserExists(id string) bool {
	r.mutx.Lock()
	defer r.mutx.Unlock()
	_, ok := r.users[id]
	return ok
}

func (r *Room) DeleteUser(id string) {
	r.mutx.Lock()
	defer r.mutx.Unlock()
	delete(r.users, id)
}

func (r *Room) GetUserCount() int {
	r.mutx.Lock()
	defer r.mutx.Unlock()
	return len(r.users)
}

func (r *Room) ForEvery(callback func(string, net.Conn)) {
	r.mutx.Lock()
	defer r.mutx.Unlock()
	for k, v := range r.users {
		callback(k, v)
	}
}

func (r *Room) Queue() {
	r.mutx.Lock()
	defer r.mutx.Unlock()
	r.queue += 1
}

func (r *Room) Dequeue() {
	r.mutx.Lock()
	defer r.mutx.Unlock()
	r.queue -= 1
}

func (r *Room) GetQueue() uint {
	r.mutx.Lock()
	defer r.mutx.Unlock()
	return r.queue
}

type RoomsType struct {
	mutx  sync.Mutex
	rooms map[string]*Room
}

func (rt *RoomsType) New() *RoomsType {
	rt.mutx = sync.Mutex{}
	rt.rooms = map[string]*Room{}
	return rt
}

func (rt *RoomsType) NewRoom(roomId string, room *Room) {
	rt.mutx.Lock()
	defer rt.mutx.Unlock()
	rt.rooms[roomId] = room
}

func (rt *RoomsType) GetRoom(roomId string) (*Room, bool) {
	rt.mutx.Lock()
	defer rt.mutx.Unlock()
	room, ok := rt.rooms[roomId]
	return room, ok
}

func (rt *RoomsType) DeleteRoom(roomId string) {
	rt.mutx.Lock()
	defer rt.mutx.Unlock()
	room, ok := rt.rooms[roomId]
	if ok {
		if room.GetQueue() == 0 {
			delete(rt.rooms, roomId)
		}
	}
}

var rooms *RoomsType

func requestUsername(conn net.Conn, room *Room) string {
	buffer := make([]byte, 64)

	for {
		n, err := conn.Read(buffer)

		if err != nil || n == 0 {
			return ""
		}

		uid := string(buffer)

		if !room.UserExists(uid) {
			fmt.Fprint(conn, "n") // new
			return uid
		}
		fmt.Fprint(conn, "e") // exist
	}
}

func joinRoom(conn net.Conn, room_id string) {
	room, ok := rooms.GetRoom(room_id)

	if !ok {
		fmt.Fprint(conn, "c") // creates new
		room = new(Room).New()
		rooms.NewRoom(room_id, room)
	} else {
		fmt.Fprintf(conn, "j%d", room.GetUserCount()) // joined to an existing one and sends members count
	}

	room.Queue()

	uid := requestUsername(conn, room)

	if uid == "" {
		return
	}

	room.AddUser(conn, uid)
	room.Dequeue()

	// reading packets from user

	for {
		buffer := make([]byte, 1024)

		n, err := conn.Read(buffer)

		if err != nil || n == 0 {
			room.DeleteUser(uid)
			if room.GetUserCount() == 0 {
				rooms.DeleteRoom(room_id)
			}
			break
		}

		room.ForEvery(func(nuid string, c net.Conn) {
			if uid != nuid {
				c.Write(buffer)
			}
		})
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buffer := make([]byte, 128)

	n, err := conn.Read(buffer)

	if err != nil || n != 128 {
		return
	}

	roomId := string(buffer)

	joinRoom(conn, roomId)
}

func main() {
	// loading certificates
	tlscert, err := tls.LoadX509KeyPair("./cert.pem", "./key.pem")

	if err != nil {
		fmt.Printf("Error while creating chain: %s", err.Error())
		return
	}

	// initializing rooms
	rooms = new(RoomsType).New()

	//starting server
	listener, err := tls.Listen("tcp", ":10500", &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{tlscert}})

	if err != nil {
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()

		if err != nil {
			continue
		}

		go handleConnection(conn)
	}
}
