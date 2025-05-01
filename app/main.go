package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		log.Fatal("Failed to connect to 0.0.0.0:6379", err.Error())
	}
	defer listener.Close()
	fmt.Println("Server is listening on port 6379...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	store := NewStore()
	for {
		resp := newResp(conn)
		value, err := resp.Read()
		if err != nil {
			fmt.Println("Error reading RESP:", err)
			return
		}

		// Check if the value is a RESP array
		if value.typ != "array" || len(value.array) < 1 {
			fmt.Println("Invalid RESP array")
			return
		}

		// Get the command
		command := value.array[0]
		if command.typ != "bulk" {
			fmt.Println("Unsupported command or invalid format")
			return
		}

		switch command.bulk {
		case "ECHO":
			handleEcho(conn, value, err)
		case "PING":
			handlePing(conn, value, err)
		case "SET":
			handleSet(conn, store, value, err)
		case "GET":
			handleGet(conn, store, value, err)
		default:
			// Unsupported command
			fmt.Println("Unsupported command:", command.bulk)
			return
		}
	}
}

func handleEcho(conn net.Conn, value Value, err error) {
	if len(value.array) < 2 {
		fmt.Println("Invalid ECHO command format")
		return
	}
	arg := value.array[1]
	if arg.typ != "bulk" {
		fmt.Println("Invalid argument for ECHO")
		return
	}
	response := fmt.Sprintf("$%d\r\n%s\r\n", len(arg.bulk), arg.bulk)
	_, err = conn.Write([]byte(response))
	if err != nil {
		fmt.Println("Error writing response:", err)
		return
	}
}

func handlePing(conn net.Conn, value Value, err error) {
	if len(value.array) < 1 {
		fmt.Println("Invalid PING command format")
		return
	}
	conn.Write([]byte("+PONG\r\n"))
	if err != nil {
		fmt.Println("Error writing response:", err)
		return
	}
}

func handleSet(conn net.Conn, store *Store, value Value, err error) {
	if len(value.array) < 3 {
		fmt.Println("Invalid SET command format")
		return
	}
	key := value.array[1]
	val := value.array[2]
	store.data[key.bulk] = val.bulk
	fmt.Println("val saved: ", store.data[key.bulk])
	if len(value.array) == 5 && value.array[3].bulk == "px" {
		exp := value.array[4]
		expiryValue, parseErr := strconv.ParseInt(exp.bulk, 10, 64)
		if parseErr != nil {
			fmt.Println("Error parsing expiration time:", parseErr)
			return
		}
		store.timeouts[key.bulk] = time.Now().Add(time.Duration(expiryValue) * time.Millisecond)
		fmt.Println(val.bulk, " expired at: ", store.timeouts[key.bulk])
	}
	conn.Write([]byte("+OK\r\n"))
	if err != nil {
		fmt.Println("Error writing response:", err)
		return
	}
}

func handleGet(conn net.Conn, store *Store, value Value, err error) {
	if len(value.array) < 2 {
		fmt.Println("Invalid GET command format")
		return
	}
	key := value.array[1]
	fmt.Print("key: ", key.bulk, "\n")
	val, ok := store.data[key.bulk]
	fmt.Println("val: ", val)
	fmt.Println("ok: ", ok)
	if !ok {
		fmt.Println("Key not found")
		conn.Write([]byte("$-1\r\n"))
		return
	}
	// Check if we have set a timeout for this key and if it has expired
	if _, ok := store.timeouts[key.bulk]; ok && time.Now().After(store.timeouts[key.bulk]) {
		fmt.Println("Key expired")
		delete(store.data, key.bulk)
		delete(store.timeouts, key.bulk)
		conn.Write([]byte("$-1\r\n"))
		return
	}
	conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)))
	if err != nil {
		fmt.Println("Error writing response:", err)
		return
	}
}
