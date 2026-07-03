package main

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"cifras/internal/types"
)

func TestMain(m *testing.M) {
	_ = LoadDictionary(bytes.NewReader(dictionaryBytes))
	m.Run()
}

// sendSync envía una acción al room y espera hasta que haya sido procesada.
func sendSync(t *testing.T, room *GameRoom, action GameAction) {
	t.Helper()
	done := make(chan struct{})
	room.ActionChan <- action
	room.ActionChan <- GameAction{Type: "SETUP", Extra: func() { close(done) }}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("sendSync timed out")
	}
}

// assertSync ejecuta fn dentro del goroutine de Run (acceso seguro al estado).
func assertSync(t *testing.T, room *GameRoom, fn func()) {
	t.Helper()
	done := make(chan struct{})
	room.ActionChan <- GameAction{Type: "SETUP", Extra: func() {
		defer close(done)
		fn()
	}}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("assertSync timed out")
	}
}

func TestGameRoomStartNewRound(t *testing.T) {
	room := NewGameRoom()
	go room.Run()
	defer room.Shutdown()

	client1 := &Client{
		ID:       "client-1",
		SendChan: make(chan types.ServerMessage, 10),
		Room:     room,
	}
	sendSync(t, room, GameAction{Type: "JOIN", PlayerID: client1.ID, Client: client1})
	assertSync(t, room, func() {
		if room.State.State != types.StateLobby {
			t.Fatalf("expected LOBBY, got %s", room.State.State)
		}
	})
	sendSync(t, room, GameAction{Type: "READY", PlayerID: client1.ID})
	assertSync(t, room, func() {
		if room.State.State != types.StatePlaying && room.State.State != types.StateChoosing {
			t.Fatalf("expected PLAYING or CHOOSING after ready, got %s", room.State.State)
		}
	})
}

func TestGameRoomSubmitCifras(t *testing.T) {
	room := NewGameRoom()
	go room.Run()
	defer room.Shutdown()

	client1 := &Client{
		ID:       "client-1",
		SendChan: make(chan types.ServerMessage, 10),
		Room:     room,
	}
	sendSync(t, room, GameAction{Type: "JOIN", PlayerID: client1.ID, Client: client1})
	sendSync(t, room, GameAction{Type: "READY", PlayerID: client1.ID})

	assertSync(t, room, func() {
		room.State.State = types.StatePlaying
		room.State.RoundType = types.RoundCifras
		room.State.Numbers = []int{2, 3, 10, 5, 1, 1}
		room.State.TargetNumber = 60
		room.BestAnswers = make(map[string]types.PlayerResult)
	})

	expr := "10 * 5 = 50\n50 + 2 = 52"
	sendSync(t, room, GameAction{
		Type:     "SUBMIT",
		PlayerID: client1.ID,
		Message:  types.ClientMessage{Expr: expr, Number: 52},
	})

	assertSync(t, room, func() {
		ans, ok := room.BestAnswers[client1.ID]
		if !ok {
			t.Fatalf("expected answer for client-1")
		}
		if ans.Distance != 8 {
			t.Fatalf("expected distance 8, got %d", ans.Distance)
		}
	})
}

func TestGameRoomFinishCifrasScoring(t *testing.T) {
	room := NewGameRoom()
	go room.Run()
	defer room.Shutdown()

	c1 := &Client{ID: "c1", SendChan: make(chan types.ServerMessage, 10), Room: room}
	c2 := &Client{ID: "c2", SendChan: make(chan types.ServerMessage, 10), Room: room}

	sendSync(t, room, GameAction{Type: "JOIN", PlayerID: c1.ID, Client: c1})
	sendSync(t, room, GameAction{Type: "JOIN", PlayerID: c2.ID, Client: c2})

	assertSync(t, room, func() {
		room.State.State = types.StatePlaying
		room.State.RoundType = types.RoundCifras
		room.State.TargetNumber = 100
		room.State.Numbers = []int{10, 10, 5, 2, 1, 1}
		room.BestAnswers = map[string]types.PlayerResult{
			c1.ID: {PlayerID: c1.ID, Name: "A", FinalNumber: 100, Distance: 0, Expression: "10 * 10 = 100"},
			c2.ID: {PlayerID: c2.ID, Name: "B", FinalNumber: 95, Distance: 5, Expression: "10 * 10 = 100\n100 - 5 = 95"},
		}
	})

	sendSync(t, room, GameAction{Type: "TIMEOUT"})

	assertSync(t, room, func() {
		if c1.Player.Score != 10 {
			t.Fatalf("expected c1 score 10, got %d", c1.Player.Score)
		}
		if c2.Player.Score != 0 {
			t.Fatalf("expected c2 score 0, got %d", c2.Player.Score)
		}
		if room.State.Winner != "A" {
			t.Fatalf("expected winner A, got %s", room.State.Winner)
		}
	})
}

func TestGameRoomFinishCifrasScoringClosest(t *testing.T) {
	room := NewGameRoom()
	go room.Run()
	defer room.Shutdown()

	c1 := &Client{ID: "c1", SendChan: make(chan types.ServerMessage, 10), Room: room}
	c2 := &Client{ID: "c2", SendChan: make(chan types.ServerMessage, 10), Room: room}

	sendSync(t, room, GameAction{Type: "JOIN", PlayerID: c1.ID, Client: c1})
	sendSync(t, room, GameAction{Type: "JOIN", PlayerID: c2.ID, Client: c2})

	assertSync(t, room, func() {
		room.State.State = types.StatePlaying
		room.State.RoundType = types.RoundCifras
		room.State.TargetNumber = 100
		room.BestAnswers = map[string]types.PlayerResult{
			c1.ID: {PlayerID: c1.ID, Name: "A", FinalNumber: 98, Distance: 2},
			c2.ID: {PlayerID: c2.ID, Name: "B", FinalNumber: 95, Distance: 5},
		}
	})

	sendSync(t, room, GameAction{Type: "TIMEOUT"})

	assertSync(t, room, func() {
		if c1.Player.Score != 7 {
			t.Fatalf("expected c1 score 7, got %d", c1.Player.Score)
		}
		if c2.Player.Score != 0 {
			t.Fatalf("expected c2 score 0, got %d", c2.Player.Score)
		}
		if room.State.Winner != "A" {
			t.Fatalf("expected winner A, got %s", room.State.Winner)
		}
	})
}

func TestGameRoomFinishLetrasScoring(t *testing.T) {
	room := NewGameRoom()
	go room.Run()
	defer room.Shutdown()

	c1 := &Client{ID: "c1", SendChan: make(chan types.ServerMessage, 10), Room: room}
	c2 := &Client{ID: "c2", SendChan: make(chan types.ServerMessage, 10), Room: room}

	sendSync(t, room, GameAction{Type: "JOIN", PlayerID: c1.ID, Client: c1})
	sendSync(t, room, GameAction{Type: "JOIN", PlayerID: c2.ID, Client: c2})

	assertSync(t, room, func() {
		room.State.State = types.StatePlaying
		room.State.RoundType = types.RoundLetras
		room.State.Letters = strings.Split("ABCDEFGHIJ", "")
		room.BestAnswers = map[string]types.PlayerResult{
			c1.ID: {PlayerID: c1.ID, Name: "A", Word: "ABCDEFGHIJ"},
			c2.ID: {PlayerID: c2.ID, Name: "B", Word: "ABCDE"},
		}
	})

	sendSync(t, room, GameAction{Type: "TIMEOUT"})

	assertSync(t, room, func() {
		if c1.Player.Score != 10 {
			t.Fatalf("expected c1 score 10, got %d", c1.Player.Score)
		}
		if c2.Player.Score != 0 {
			t.Fatalf("expected c2 score 0, got %d", c2.Player.Score)
		}
		if room.State.Winner != "A" {
			t.Fatalf("expected winner A, got %s", room.State.Winner)
		}
	})
}

func TestGameRoomCheckAllReadyTimeout(t *testing.T) {
	room := NewGameRoom()
	go room.Run()
	defer room.Shutdown()

	c1 := &Client{ID: "c1", SendChan: make(chan types.ServerMessage, 10), Room: room}
	sendSync(t, room, GameAction{Type: "JOIN", PlayerID: c1.ID, Client: c1})

	assertSync(t, room, func() {
		room.checkAllReady()
	})

	assertSync(t, room, func() {
		if room.timer != nil {
			t.Fatalf("expected no timer with 1 player not ready")
		}
	})

	sendSync(t, room, GameAction{Type: "READY", PlayerID: c1.ID})

	assertSync(t, room, func() {
		if room.timer == nil {
			t.Fatalf("expected timer started with 1 player ready")
		}
	})
}
