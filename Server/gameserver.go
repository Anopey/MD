package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

var activeGames = make([]*game, 0, 50)

func initializeGameServer(p1, p2 *player) {
	gameInstance := game{
		p1:              p1,
		p2:              p2,
		p1ready:         false,
		p2ready:         false,
		p1tempo:         0.0,
		p2tempo:         0.0,
		currentPhase:    initializing,
		gameCommandChan: make(chan *playerMessage, 4),
	}
	p1.activeGame = &gameInstance
	p2.activeGame = &gameInstance

	writeToPlayer(p1, "MD GAME-INIT "+"~"+p2.name+"\n")
	writeToPlayer(p2, "MD GAME-INIT "+"~"+p1.name+"\n")

	tendGameChannel(&gameInstance)
}

func tendGameChannel(g *game) {
	for g != nil {
		select {
		case playerMessage := <-g.gameCommandChan:
			var inputVar float32 = 0
			fields := strings.Fields(playerMessage.msg[:len(playerMessage.msg)-1])
			if len(fields) == 4 {
				println("SERVER RECIEVED FLOAT ARGUMENT FOR GAME: " + playerMessage.msg)
				out, err := strconv.ParseFloat(fields[3], 32)
				if err != nil {
					log.Println("ERROR: GAME INPUT VARIABLE IS NOT PARSEABLE TO FLOAT32!")
				}
				inputVar = float32(out)
				playerMessage.msg = fields[0] + " " + fields[1] + " " + fields[2]
			}

			//signals not followed by \n have input available to them ;)
			switch playerMessage.msg {
			case "MD GAME INNER-PLAYER-DISCONNECT\n":
				if playerMessage.p == g.p1 {
					writeToPlayer(g.p2, "MD GAME FDISCONNECT\n")
				} else if playerMessage.p == g.p2 {
					writeToPlayer(g.p1, "MD GAME FDISCONNECT\n")
				} else {
					fmt.Println("ERROR: NEITHER OF THE PLAYERS ARE EQUIVALENT TO THE OWNER OF THE FDISCONNECT MESSAGE SENT TO THIS GAME INSTANCE")
				}
				break
			case "MD GAME READY":
				if g.currentPhase != initializing {
					fmt.Println("ERROR: A PLAYER HAS ATTEMPTED TO READY UP DESPITE THE GAME NOT BEING IN THE INITIALIZATION PHASE")
					break
				}
				if playerMessage.p == g.p1 {
					g.p1ready = true
					g.p1tempo = inputVar
				} else if playerMessage.p == g.p2 {
					g.p2ready = true
					g.p2tempo = inputVar
				} else {
					fmt.Println("ERROR: NEITHER OF THE PLAYERS ARE EQUIVALENT TO THE OWNER OF THE READY MESSAGE SENT TO THIS GAME INSTANCE")
				}

				if g.p1ready && g.p2ready {
					go gameTempoProcess(g)
					//left here
				}
			}
			break
		}
	}
}

func gameTempoProcess(g *game) {
	fmt.Println("STARTING TEMPO PROCESS FOR " + g.p1.name + " and " + g.p2.name)
}
