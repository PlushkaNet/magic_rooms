package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Room struct {
	mutx  sync.Mutex
	users map[string]net.Conn
	queue uint
}

func NewRoom() *Room {
	room := Room{}
	room.mutx = sync.Mutex{}
	room.users = map[string]net.Conn{}
	return &room
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

func NewRoomsType() *RoomsType {
	rt := RoomsType{}
	rt.mutx = sync.Mutex{}
	rt.rooms = map[string]*Room{}
	return &rt
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
		room = NewRoom()
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

func parseArguments(args []string) map[string]string {
	kwargs := map[string]string{}
	argsLen := len(args)

	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			if (argsLen - i) > 1 {
				kwargs[strings.TrimPrefix(arg, "-")] = args[i+1]
			}
		}
	}

	return kwargs
}

func main() {
	// initializing rooms
	rooms = NewRoomsType()

	// parsing kwargs
	kwargs := parseArguments(os.Args)

	port := kwargs["p"]

	if port == "" {
		log.Println("No port specified, using standart 10500")
		port = ":10500"
	} else {
		_, err := strconv.Atoi(port) // checking that port has valid int type

		if err != nil {
			log.Fatalln("Invalid port specified")
		}

		log.Printf("Using explicitly specified port %s\n", port)

		port = ":" + port
	}

	var listener net.Listener

	// setting up TLS server
	if kwargs["m"] == "tls" {
		cert := kwargs["cert"]
		key := kwargs["key"]

		if cert == "" || key == "" {
			log.Fatalln("You must specify -cert and -key to run in TLS mode")
		}

		log.Println("Trying to load certificates")

		pair, err := tls.LoadX509KeyPair(cert, key)

		if err != nil {
			log.Fatalln("Cannot load specified certificates")
		}

		log.Println("Starting secured server")

		listener, err = tls.Listen("tcp", port, &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pair}})

		if err != nil {
			log.Fatalf("Failed to start secured server: %s", err.Error())
		}
	} else {
		// else setting up normal tcp server
		log.Println("Starting server")
		var err error
		listener, err = net.Listen("tcp", port)
		if err != nil {
			log.Fatalf("Failed to start server: %s", err.Error())
		}
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
