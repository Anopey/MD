package main

import (
	"bufio"
	"container/list"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type queueType int
type gamePhase int

const (
	codeBased queueType = 0
	free      queueType = 1

	initializing gamePhase = 0
	active       gamePhase = 1

	timeOutMessagesSendTimeSecond time.Duration = time.Second * 5
)

type player struct {
	conn                    *net.Conn
	name                    string
	activeGame              *game
	lastMsgRecieve          time.Time
	active                  bool
	id                      int64
	writeChannel            chan *writeRequest
	disconnectClientChannel chan interface{}
	m                       sync.RWMutex
	tendedPlayersElement    *list.Element
}

type game struct {
	musicName       string
	p1              *player
	p2              *player
	p1ready         bool
	p2ready         bool
	p1tempo         float32
	p2tempo         float32
	currentPhase    gamePhase
	gameCommandChan chan *playerMessage
}

type playerMessage struct {
	p   *player
	msg string
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

	go queueSystem()
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
	go tendToClientChannels(p)
	tendToClientRead(p, scanner)
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

var tendedPlayers *list.List = list.New()

var tendedPlayersMutex sync.RWMutex = sync.RWMutex{}

//tendToClientRead handles all the reading operations relating to a specific client
func tendToClientRead(p *player, scanner *bufio.Scanner) {
	p.lastMsgRecieve = time.Now() //so no timeout occurs immediately
	tendedPlayersMutex.Lock()
	p.tendedPlayersElement = tendedPlayers.PushBack(p)
	tendedPlayersMutex.Unlock()
	for scanner.Scan() && p.active && serverActive {
		p.m.Lock()
		if !p.active {
			return
		}
		ln := scanner.Text()
		fmt.Println(p.name + ": " + ln)

		//check for game messages
		p.lastMsgRecieve = time.Now()
		gameChecked := "MD GAME"
		if ln[:len(gameChecked)] == gameChecked {
			//if starts with MD GAME:
			if p.activeGame == nil {
				fmt.Println("Recieved game message despite not being in-game from " + p.name + ": " + ln)
				p.writeChannel <- &writeRequest{
					message: "MD GAME-INVALID\n",
				}
				p.disconnectClientChannel <- struct{}{}
				return
			}
			p.activeGame.gameCommandChan <- &playerMessage{
				p:   p,
				msg: ln,
			}
		}

		//deal with non-game messages
		switch ln {
		case "MD CLOSE\n":
			//handle this player being no more.
			p.disconnectClientChannel <- struct{}{}
		case "MD NO TIMEOUT\n":
			break
		case "MD ENQUEUE\n":
			queuedPlayersChannel <- p
			break
		default:
			fmt.Println("Recieved unknown message from " + p.name + ": " + ln)
			p.writeChannel <- &writeRequest{
				message: "MD INVALID\n",
			}
			p.disconnectClientChannel <- struct{}{}
			return
		}
		p.m.Unlock()
	}
	fmt.Println("disconnecting " + p.name)
	p.disconnectClientChannel <- struct{}{}
}

//tendToClientChannels ensures that only one routine per client can tend to the players' channels, such as writing
func tendToClientChannels(p *player) {
	conn := *p.conn
	for p.active && serverActive {
		p.m.Lock()
		if !p.active {
			return
		}
		select {
		case w := <-p.writeChannel:
			fmt.Fprint(conn, w.message)
			break
		case <-p.disconnectClientChannel:
			fmt.Println("disconnecting through channel: " + p.name)
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

	p.active = false

	//remove from tended players
	tendedPlayersMutex.Lock()
	tendedPlayers.Remove(p.tendedPlayersElement)
	tendedPlayersMutex.Unlock()

	//deal with active games
	if p.activeGame != nil {
		p.activeGame.gameCommandChan <- &playerMessage{
			p:   p,
			msg: "MD GAME INNER-PLAYER-DISCONNECT\n",
		}
	}
}

func timeoutRoutine() {
	for serverActive {
		time.Sleep(timeOutMessagesSendTimeSecond)
		tendedPlayersMutex.RLock()
		toRemove := make([]*player, 0, tendedPlayers.Len()/10)
		ele := tendedPlayers.Front()
		for ele != nil {
			p := ele.Value.(*player)
			if p.lastMsgRecieve.Add(timeOutMessagesSendTimeSecond * 3).Before(time.Now()) {
				toRemove = append(toRemove, p)
				continue
			}
			p.writeChannel <- &writeRequest{
				message: "MD NO TIMEOUT\n",
			}
			ele = ele.Next()
		}
		tendedPlayersMutex.RUnlock()

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
