package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type queueType int

const (
	codeBased queueType = 0
	free      queueType = 1

	timeOutMessagesSendTimeSecond time.Duration = time.Second * 5
)

type player struct {
	conn                    *net.Conn
	name                    string
	activeQueue             *queue
	activeGame              *game
	lastMsgRecieve          time.Time
	active                  bool
	id                      int64
	writeChannel            chan *writeRequest
	disconnectClientChannel chan interface{}
	m                       sync.RWMutex
}

type game struct {
	musicName string
	p1        player
	p2        player
}

type queue struct {
	qType queueType
}

var lastID int64 = 0
var serverActive bool = false

func main() {
	li, err := net.Listen("tcp", ":52515")
	if err != nil {
		log.Fatalln(err.Error())
	}

	fmt.Println("Now listening on port 52515...")

	serverActive = true

	go gameServer()
	go delegateChannels()
	go timeoutRoutine()

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
	p := handleInitialConnection(conn, scanner)
	if p == nil {
		fmt.Println(time.Now().Format("2006-01-02 15:04:05") + ": " + "INVALID REQUEST FROM: " + (*conn).RemoteAddr().String())
		fmt.Fprint(*conn, "MD INVALID\n") //HOW DARE YOU NOT USE THE MD PROTOCOL. DAMN HTTP NERDS
		(*conn).Close()
		return
	}
	//ok player is created and has connection
	fmt.Fprint(*conn, "MD OK\n")
	tendToClientRead(p, scanner)
	go tendToClientChannels(p, scanner)
}

func handleInitialConnection(conn *net.Conn, scanner *bufio.Scanner) *player {
	fmt.Println("handling initial player connection...")
	if !(*scanner).Scan() {
		fmt.Println("Empty connection request detected.")
		(*conn).Close()
		return nil
	}
	fields, flag := parseUtilsAndSignal((*scanner).Text(), 2)
	fmt.Println(fields)
	if flag != ok {
		return nil
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05") + ": " + "****NEW PLAYER: " + (*conn).RemoteAddr().String() + " " + fields[1])
	var newPlayer = player{
		conn:                    conn,
		name:                    fields[1],
		id:                      lastID,
		active:                  true,
		writeChannel:            make(chan *writeRequest, 5),
		disconnectClientChannel: make(chan interface{}, 5),
		m:                       sync.RWMutex{},
	}
	lastID++
	return &newPlayer
}

var tendedPlayers = make([]*player, 0, 50)

//tendToClientRead handles all the reading operations relating to a specific client
func tendToClientRead(p *player, scanner *bufio.Scanner) {
	tendedPlayers = append(tendedPlayers, p)
	for scanner.Scan() && p.active && serverActive {
		p.m.Lock()
		ln := scanner.Text()
		p.lastMsgRecieve = time.Now()
		switch ln {
		case "MD CLOSE\n":
			//handle this player being no more.
			p.disconnectClientChannel <- struct{}{}
		case "MD NO TIMEOUT\n":
			break
		}
		p.m.Unlock()
	}
}

//tendToClientChannels ensures that only one routine per client can tend to the players' channels, such as writing
func tendToClientChannels(p *player, scanner *bufio.Scanner) {
	conn := *p.conn
	for p.active && serverActive {
		p.m.Lock()
		select {
		case w := <-p.writeChannel:
			fmt.Fprint(conn, w.message)
			break
		case <-p.disconnectClientChannel:
			fmt.Fprint(*p.conn, "MD CLOSE\n")
			disconnectAndRemoveClient(p)
			conn.Close()
			return
		}
		p.m.Unlock()
	}
}

func disconnectAndRemoveClient(p *player) {
	//deal with all that is necessary here, such as removing from tended etc.

}

func timeoutRoutine() {
	for serverActive {
		time.Sleep(timeOutMessagesSendTimeSecond)
		toRemove := make([]*player, 0, len(tendedPlayers)/10)
		for _, v := range tendedPlayers {
			if v.lastMsgRecieve.Add(timeOutMessagesSendTimeSecond * 3).Before(time.Now()) {
				toRemove = append(toRemove, v)
				continue
			}
			fmt.Fprintln(v.name + " reee")
			v.writeChannel <- &writeRequest{
				message: "MD NO TIMEOUT\n",
			}
		}

		//now for removal
		for _, v := range toRemove {
			v.disconnectClientChannel <- struct{}{}
		}
	}
}

//serverCloseChannel is called when server should close.
var serverCloseChannel = make(chan interface{})

func delegateChannels() {
	for serverActive {
		select {
		case <-serverCloseChannel:

			break
		}
	}
}

type writeRequest struct {
	message string
}

func handleWriteRequest(conn *net.Conn, req *writeRequest) {
	fmt.Fprint(*conn, req.message)
}
