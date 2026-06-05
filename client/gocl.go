package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const PROTOCOL_V int = 1
const symbols string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const BUFFER_SIZE uint = 1024

func generateRandomString(n int, r *rand.Rand) string {
	result := ""
	for range n {
		result += string(symbols[r.Intn(62)])
	}
	return result
}

func listenForMessages(conn net.Conn) {
	for {
		buffer := make([]byte, BUFFER_SIZE)
		n, err := conn.Read(buffer)

		if err != nil || n == 0 {
			fmt.Printf("Server unexpectedly closed connection: %s\n", err.Error())
			return
		}

		fmt.Printf("Roommate: %s\n", string(buffer))
	}
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
	kwargs := parseArguments(os.Args)

	var address string
	if kwargs["addr"] != "" {
		address = kwargs["addr"]
		fmt.Printf("addr specified, using address: %s\n", address)
	} else {
		fmt.Printf("What address we should connect?\n")
		_, err := fmt.Scan(&address)
		if err != nil {
			fmt.Printf("Error occured while reading input: %s\n", err.Error())
			return
		}
		if address == "" {
			fmt.Println("Error: address must be specified")
			return
		}
	}

	fmt.Printf("Trying to connect to %s\n", address)

	var conn net.Conn
	var err error

	if kwargs["m"] == "tls" {
		fmt.Println("Connection mode switched to TLS")

		conn, err = tls.Dial("tcp", address, &tls.Config{MinVersion: tls.VersionTLS12})
	} else {
		fmt.Print("\n! Using basic insecure mode\n! You can switch to TLS mode using -m tls flag in command line arguments\n\n")

		conn, err = net.Dial("tcp", address)
	}

	if err != nil {
		fmt.Println("Error while connecting to the server")
		return
	}

	defer conn.Close()

	// err = conn.SetDeadline(time.Time{}.Add(time.Second * 10))

	// if err != nil {
	// 	fmt.Printf("Cannot set timeout timer (deadline): %s", err.Error())
	// 	return
	// }

	fmt.Printf("Connected successfully;\nremote address: %s\nlocal  address: %s\n\n", conn.RemoteAddr().String(), conn.LocalAddr().String())
	fmt.Printf("Using protocol version %d\n", PROTOCOL_V)

	fmt.Print("Do you know your room? (y/N): ")

	var choice string
	_, err = fmt.Scan(&choice)

	if err != nil {
		fmt.Printf("Error while reading CMDIN: %s\n", err.Error())
		return
	}

	if len(choice) > 0 && choice[0] == 'y' {
		for {
			fmt.Print("Enter your room id: ")

			choice = ""
			_, err = fmt.Scan(&choice)

			if err != nil {
				fmt.Printf("Error while reading CMDIN: %s\n", err.Error())
				return
			}

			if len(choice) == 128 {
				fmt.Println("Right length")
				break
			} else {
				fmt.Println("Incorrect length; protocol means room id with 128 symbols")
			}
		}
	} else {
		fmt.Println("Generating new room ID...")
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		choice = generateRandomString(128, r)
	}

	fmt.Printf("Connecting to room %s\n", choice)

	_, err = fmt.Fprint(conn, choice)

	if err != nil {
		fmt.Printf("Error while connecting to room (transfering packet): %s\n", err.Error())
		return
	}

	buffer := make([]byte, 16)
	n, err := conn.Read(buffer)

	if err != nil || n == 0 {
		fmt.Printf("Error while waiting response from the server: %s\n", err.Error())
		return
	}

	streply := string(bytes.Trim(buffer, "\x00"))

	switch streply[0] {
	case 'c':
		fmt.Println("This room wasn't exist, created new")
	case 'j':
		fmt.Print("Joined existing room ")
		// idk why it converts, just for sure that it is number
		userCount, err := strconv.Atoi(strings.Replace(streply, "j", "", 1))

		if err != nil {
			fmt.Printf("\nError while parsing userCount, proceeding without it: %s\n", err.Error())
		} else {
			fmt.Printf("(members online: %d)\n", userCount)
		}
	default:
		fmt.Println("Got malformed response from the server")
		return
	}

	fmt.Println("Now let's setup your username")

	for {
		fmt.Print("Enter your new username: ")
		choice = ""

		_, err = fmt.Scan(&choice)

		if err != nil {
			fmt.Printf("Error while reading CMDIN: %s\n", err.Error())
			return
		}

		fmt.Println("Checking that this username avaliable on server")
		_, err = conn.Write([]byte(choice))

		if err != nil {
			fmt.Printf("Error while checking username (writing): %s\n", err.Error())
			return
		}

		buffer := make([]byte, 4)
		n, err = conn.Read(buffer)

		if err != nil || n == 0 {
			fmt.Printf("Error while reading response from server: %s\n", err.Error())
			return
		}

		if string(buffer)[0] == 'e' {
			fmt.Printf("Username %s does already claimed, try another one\n", choice)
		} else {
			fmt.Printf("Success! Claimed username %s\n", choice)
			break
		}
	}

	fmt.Print("Starting service for reading for new messages...\n\nNow you can write messages\n\n")
	go listenForMessages(conn)

	var message string
	reader := bufio.NewReader(os.Stdin)

	for {
		message, err = reader.ReadString('\n')

		if err != nil {
			fmt.Printf("Exiting: %s\n", err.Error())
			return
		}

		message, _ = strings.CutSuffix(message, "\n") // rewrites message with its copy without trailing \n

		_, err = fmt.Fprint(conn, message)

		if err != nil {
			fmt.Printf("Message wasn't sent: %s", err.Error())
		}
	}
}
