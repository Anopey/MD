package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

type queueType int

const (
	codeBased queueType = 0
	free      queueType = 1
)

type player struct {
	conn        *net.Conn
	name        string
	activeQueue *queue
	activeGame  *game
}

type game struct {
	musicName string
	p1        player
	p2        player
}

type queue struct {
	qType queueType
}

func main() {
	li, err := net.Listen("tcp", ":52515")
	if err != nil {
		log.Fatalln(err.Error())
	}

	fmt.Println("Now listening on port 52515...")

	go gameServer()

	for {
		conn, err := li.Accept()
		if err != nil {
			log.Fatalln(err.Error())
		}
		fmt.Println("New connection from " + conn.RemoteAddr().String())
		go handleGameConnection(&conn)

	}
}

func handleGameConnection(conn *net.Conn) {
	scanner := bufio.NewScanner(*conn)
	if !scanner.Scan() {
		fmt.Println("Empty connection request detected.")
		(*conn).Close()
		return
	}
	p := handleInitialConnection(conn, scanner)
	if p == nil {
		fmt.Println(time.Now().Format("2006-01-02 15:04:05") + ": " + "INVALID REQUEST FROM: " + (*conn).RemoteAddr().String())
		fmt.Fprint(*conn, "MD INVALID") //HOW DARE YOU NOT USE THE MD PROTOCOL. DAMN HTTP NERDS
		(*conn).Close()
		return
	}
	//ok player is created and has connection
	fmt.Fprint(*conn, "MD OK")
}

func handleInitialConnection(conn *net.Conn, scanner *bufio.Scanner) *player {
	fields, flag := parseUtilsAndSignal(scanner.Text(), 2)
	if flag != ok {
		return nil
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05") + ": " + "****NEW PLAYER: " + (*conn).RemoteAddr().String() + " " + fields[1])
	var newPlayer = player{
		conn: conn,
		name: fields[1],
	}
	return &newPlayer
}
