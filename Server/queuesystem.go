package main

import (
	"fmt"
	"strconv"
	"time"
)

const (
	queueMessageSendCooldown time.Duration = time.Second * 1
)

var queuedPlayersChannel = make(chan *player, 5)
var queuedPlayers = make([]*player, 0, 2)

func queueSystem() {
	for serverActive {
		select {
		case newPlayer := <-queuedPlayersChannel:
			if newPlayer.activeGame != nil {
				newPlayer.writeChannel <- &writeRequest{
					message: "MD INVALID\n",
				}
				break
			}
			go handleQueuedPlayer(newPlayer)
			break
		}
	}
}

func handleQueuedPlayer(newPlayer *player) {
	queuedPlayers = append(queuedPlayers, newPlayer)
	if len(queuedPlayers) >= 2 {
		go initializeGameServer(queuedPlayers[0], queuedPlayers[1])

		//remove from array

		// queuedPlayers[0] = queuedPlayers[len(queuedPlayers)-1]
		// queuedPlayers[len(queuedPlayers)-1] = nil
		// queuedPlayers[1] = queuedPlayers[len(queuedPlayers)-2]
		// queuedPlayers[len(queuedPlayers)-2] = nil
		// queuedPlayers = queuedPlayers[:len(queuedPlayers)-2]

		queuedPlayers = queuedPlayers[2:]

	} else {
		for newPlayer.activeGame == nil {
			tendedPlayersMutex.RLock()
			newPlayer.writeChannel <- &writeRequest{
				message: "MD QUEUE " + strconv.Itoa(tendedPlayers.Len()) + "\n",
			}
			tendedPlayersMutex.RUnlock()
			fmt.Println("tended players read for sending queue data to: " + newPlayer.name)
			time.Sleep(queueMessageSendCooldown)
		}
	}
}
