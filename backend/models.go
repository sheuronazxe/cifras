package main

import (
	"cifras/internal/types"

	"github.com/coder/websocket"
)

type GameAction struct {
	PlayerID string
	Type     string // "JOIN", "LEAVE", "READY", "NAME", "CHOOSE_VOWELS", "SUBMIT", "TIMEOUT", "SETUP"
	Message  types.ClientMessage
	Conn     *websocket.Conn
	Client   *Client
	RoundGen int64 // prevents stale TIMEOUT events
	Extra    any   // used internally for test setup and timer coordination
}
