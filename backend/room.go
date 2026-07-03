package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"cifras/internal/types"

	"github.com/coder/websocket"
)

var stepRegex = regexp.MustCompile(`^(\d+)\s*([+\-*/])\s*(\d+)\s*=\s*(\d+)$`)

const (
	NumbersRoundDuration = 50 * time.Second
	LettersRoundDuration = 50 * time.Second
	ChooserTimeout       = 10 * time.Second
	ReadyTimeout         = 30 * time.Second
	MaxClients           = 20
)

type GameRoom struct {
	Clients           map[string]*Client
	ActionChan        chan GameAction
	State             types.SyncData
	BestAnswers       map[string]types.PlayerResult
	HistoricalPlayers map[string]types.Player

	timer      *time.Timer
	chooserIdx int
	timerGen   int64
	done       chan struct{}
	runDone    chan struct{} // closed when Run() exits
	closed     atomic.Bool
}

func NewGameRoom() *GameRoom {
	return &GameRoom{
		Clients:    make(map[string]*Client),
		ActionChan: make(chan GameAction, 256),
		State: types.SyncData{
			State:        types.StateLobby,
			CurrentRound: 0,
		},
		BestAnswers:       make(map[string]types.PlayerResult),
		HistoricalPlayers: make(map[string]types.Player),
		done:              make(chan struct{}),
		runDone:           make(chan struct{}),
	}
}

func (r *GameRoom) Run() {
	defer close(r.runDone)
	for action := range r.ActionChan {
		switch action.Type {
		case "JOIN":
			r.handleJoin(action)
		case "LEAVE":
			r.handleLeave(action)
		case "READY":
			r.handleReady(action)
		case "NAME":
			r.handleName(action)
		case "CHOOSE_VOWELS":
			r.handleChooseVowels(action)
		case "SUBMIT":
			r.handleSubmit(action)
		case "TIMEOUT": // Internal event
			r.handleTimeout(action)
		case "SETUP": // Only used in tests
			if fn, ok := action.Extra.(func()); ok {
				fn()
			}
		}
	}
}

func (r *GameRoom) Shutdown() {
	r.closed.Store(true)
	close(r.done)
	close(r.ActionChan)
	<-r.runDone // wait for Run() to exit before accessing r.timer
	if r.timer != nil {
		r.timer.Stop()
	}
}

// safeSendAction sends an action to ActionChan, recovering if the channel is closed.
func (r *GameRoom) safeSendAction(action GameAction) {
	if r.closed.Load() {
		return
	}
	defer func() { recover() }()
	r.ActionChan <- action
}

func (r *GameRoom) broadcastState() {
	// Rebuild players list
	var players []types.Player
	for _, c := range r.Clients {
		players = append(players, c.Player)
	}
	sort.Slice(players, func(i, j int) bool {
		if players[i].Score != players[j].Score {
			return players[i].Score > players[j].Score // Descending
		}
		return players[i].ID < players[j].ID // Deterministic secondary sorting (ID)
	})
	r.State.Players = players
	r.State.ServerTime = time.Now().UnixMilli()

	msg := types.ServerMessage{Type: "SYNC", Payload: r.State}
	for id, c := range r.Clients {
		select {
		case c.SendChan <- msg:
		default:
			// Full channel: client is too slow, disconnect it
			log.Printf("SendChan full for %s, disconnecting", id)
			close(c.SendChan)
			delete(r.Clients, id)
			c.closeConn()
		}
	}
}

func (r *GameRoom) sendToast(clientID string, toast types.ToastMessage) {
	if c, ok := r.Clients[clientID]; ok {
		msg := types.ServerMessage{Type: "TOAST", Payload: toast}
		select {
		case c.SendChan <- msg:
		default:
		}
	}
}

func (r *GameRoom) handleJoin(action GameAction) {
	if len(r.Clients) >= MaxClients {
		action.Conn.Close(websocket.StatusPolicyViolation, "sala llena")
		return
	}

	oldClient := r.Clients[action.PlayerID]
	newClient := action.Client

	if oldClient != nil {
		// Reconnect: copy player state and clean up old connection
		newClient.Player = oldClient.Player
		oldClient.closeConn()
		close(oldClient.SendChan)
	} else if histPlayer, ok := r.HistoricalPlayers[action.PlayerID]; ok {
		// Reconnect: copy historical player state
		newClient.Player = histPlayer
		newClient.Player.IsReady = false
	} else {
		// New player
		baseName := fmt.Sprintf("Jugador %s", action.PlayerID)
		newClient.Player = types.Player{
			ID:   action.PlayerID,
			Name: r.uniqueName(baseName),
		}
	}

	r.Clients[action.PlayerID] = newClient
}

func (r *GameRoom) uniqueName(name string) string {
	existing := make(map[string]struct{}, len(r.Clients))
	for _, c := range r.Clients {
		existing[c.Player.Name] = struct{}{}
	}
	baseName := name
	for i := 1; ; i++ {
		if _, ok := existing[name]; !ok {
			return name
		}
		name = fmt.Sprintf("%s#%d", baseName, i)
	}
}

func (r *GameRoom) handleLeave(action GameAction) {
	if client, ok := r.Clients[action.PlayerID]; ok {
		// Evitar borrar el cliente si ya ha sido reemplazado por una nueva conexión
		if action.Client != nil && action.Client != client {
			return
		}

		// Guardar estado histórico antes de borrar
		r.HistoricalPlayers[action.PlayerID] = client.Player

		delete(r.Clients, action.PlayerID)
		close(client.SendChan)

		// If choosing and it's chooser, auto pick (startLettersRound broadcasts)
		if r.State.State == types.StateChoosing && r.State.ChooserID == client.Player.ID {
			r.startLettersRound(rand.IntN(4) + 3) // 3 to 6 vowels
			return
		}

		if len(r.Clients) == 0 {
			r.resetToLobby()
		} else {
			r.checkAllReady()
		}
		r.broadcastState()
	}
}

func sanitizeName(name string) string {
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, name)
	return strings.TrimSpace(name)
}

func (r *GameRoom) handleName(action GameAction) {
	if c, ok := r.Clients[action.PlayerID]; ok {
		newName := sanitizeName(action.Message.Name)
		if newName != "" {
			runes := []rune(newName)
			if len(runes) > 20 {
				newName = string(runes[:20])
			}
			if newName != c.Player.Name {
				c.Player.Name = r.uniqueName(newName)
			}
		}
		
		// Send WELCOME message back to this client containing their unique ID and Name
		welcomeMsg := types.ServerMessage{
			Type:    "WELCOME",
			Payload: c.Player,
		}
		select {
		case c.SendChan <- welcomeMsg:
		default:
		}
		
		r.broadcastState()
	}
}

func (r *GameRoom) handleReady(action GameAction) {
	if r.State.State != types.StateLobby && r.State.State != types.StateFinished {
		return
	}
	if c, ok := r.Clients[action.PlayerID]; ok {
		c.Player.IsReady = !c.Player.IsReady
		r.checkAllReady()
		r.broadcastState()
	}
}

func (r *GameRoom) checkAllReady() {
	if r.State.State != types.StateLobby && r.State.State != types.StateFinished {
		return
	}

	numClients := len(r.Clients)
	if numClients == 0 {
		return
	}
	
	allReady := true
	anyReady := false
	for _, c := range r.Clients {
		if !c.Player.IsReady {
			allReady = false
		} else {
			anyReady = true
		}
	}

	if allReady {
		r.startNewRound()
	} else if anyReady && r.timer == nil {
		r.setTimer(ReadyTimeout)
	} else if !anyReady && r.timer != nil {
		r.timer.Stop()
		r.timer = nil
		r.State.EndTime = 0
	}
}

func (r *GameRoom) setTimer(d time.Duration) {
	if r.timer != nil {
		r.timer.Stop()
	}
	r.timerGen++
	gen := r.timerGen
	r.timer = time.AfterFunc(d, func() {
		defer func() { recover() }()
		select {
		case <-r.done:
			return
		case r.ActionChan <- GameAction{Type: "TIMEOUT", TimerGen: gen}:
		}
	})
	r.State.EndTime = time.Now().Add(d).UnixMilli()
}

func (r *GameRoom) handleTimeout(action GameAction) {
	if action.TimerGen != r.timerGen {
		return
	}

	// Para el estado de juego, somos más permisivos para evitar bloqueos
	if r.State.State == types.StatePlaying {
		r.finishRound()
		return
	}

	switch r.State.State {
	case types.StateLobby, types.StateFinished:
		r.startNewRound()
	case types.StateChoosing:
		r.startLettersRound(rand.IntN(4) + 3)
	}
}

func (r *GameRoom) startNewRound() {
	r.BestAnswers = make(map[string]types.PlayerResult)
	r.State.Chooser = ""
	r.State.ChooserID = ""
	// Limpiar campos de la ronda anterior para evitar estado stale
	r.State.Winner = ""
	r.State.Solution = ""
	r.State.SolutionSteps = nil
	r.State.ExactSolutionSteps = nil
	r.State.OtherResults = nil

	// Reset ready state
	for _, c := range r.Clients {
		c.Player.IsReady = false
	}

	// Pick chooser (validación temprana para Cifras y Letras)
	var ids []string
	for id := range r.Clients {
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}

	r.State.CurrentRound++

	if r.State.CurrentRound%2 != 0 {
		// Cifras
		r.State.State = types.StatePlaying
		r.State.RoundType = types.RoundCifras
		r.State.Numbers = generateNumbers()
		r.State.TargetNumber = rand.IntN(899) + 101
		r.setTimer(NumbersRoundDuration)
	} else {
		// Letras (Choosing phase)
		r.State.State = types.StateChoosing
		r.State.RoundType = types.RoundLetras

		sort.Strings(ids)
		r.chooserIdx = (r.chooserIdx + 1) % len(ids)
		chooser := r.Clients[ids[r.chooserIdx]]
		r.State.Chooser = chooser.Player.Name
		r.State.ChooserID = chooser.Player.ID

		r.setTimer(ChooserTimeout)
	}
	r.broadcastState()
}

func (r *GameRoom) handleChooseVowels(action GameAction) {
	if r.State.State != types.StateChoosing {
		return
	}
	c, ok := r.Clients[action.PlayerID]
	if !ok || c.Player.ID != r.State.ChooserID {
		return
	}
	vowels := action.Message.Vowels
	if vowels < 3 { vowels = 3 }
	if vowels > 6 { vowels = 6 }

	r.startLettersRound(vowels)
}

func (r *GameRoom) startLettersRound(vowelsCount int) {
	r.State.State = types.StatePlaying
	r.State.Letters = generateLetters(vowelsCount)
	r.setTimer(LettersRoundDuration)
	r.broadcastState()
}

func (r *GameRoom) handleSubmit(action GameAction) {
	if r.State.State != types.StatePlaying {
		return
	}
	c, ok := r.Clients[action.PlayerID]
	if !ok {
		return
	}

	if r.State.RoundType == types.RoundCifras {
		r.handleSubmitCifras(c, action.Message)
	} else {
		r.handleSubmitLetras(c, action.Message)
	}
}

func (r *GameRoom) handleSubmitCifras(c *Client, msg types.ClientMessage) {
	if msg.Expr == "" || len(msg.Expr) > 500 {
		return
	}

	steps := strings.Split(msg.Expr, "\n")
	if len(steps) > 20 {
		r.sendToast(c.ID, types.ToastMessage{Message: "Demasiados pasos (max 20)", Type: "error"})
		return
	}

	available := make(map[int]int)
	for _, n := range r.State.Numbers {
		available[n]++
	}

	lastResult := 0
	for _, step := range steps {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}

		matches := stepRegex.FindStringSubmatch(step)
		if matches == nil {
			r.sendToast(c.ID, types.ToastMessage{Message: "Formato de paso inválido", Type: "error"})
			return
		}

		a, _ := strconv.Atoi(matches[1])
		b, _ := strconv.Atoi(matches[3])
		result, _ := strconv.Atoi(matches[4])
		op := matches[2]

		if available[a] <= 0 || available[b] <= 0 {
			r.sendToast(c.ID, types.ToastMessage{Message: "Número no disponible", Type: "error"})
			return
		}
		available[a]--
		available[b]--

		expected, ok := computeOperation(a, b, op)
		if !ok || expected <= 0 || expected != result {
			r.sendToast(c.ID, types.ToastMessage{Message: "Operación inválida", Type: "error"})
			return
		}
		available[result]++
		lastResult = result
	}

	if lastResult == 0 || lastResult != msg.Number || msg.Number < 1 || msg.Number > 9999 {
		r.sendToast(c.ID, types.ToastMessage{Message: "Resultado final inválido", Type: "error"})
		return
	}

	diff := int(math.Abs(float64(r.State.TargetNumber - msg.Number)))
	prev, hasPrev := r.BestAnswers[c.ID]
	if !hasPrev || diff <= prev.Distance {
		r.BestAnswers[c.ID] = types.PlayerResult{
			PlayerID:    c.ID,
			Name:        c.Player.Name,
			FinalNumber: msg.Number,
			Distance:    diff,
			Expression:  msg.Expr,
		}
		r.sendToast(c.ID, types.ToastMessage{
			Message: fmt.Sprintf("Resultado guardado: %d (diferencia: %d)", msg.Number, diff),
			Type:    "success",
		})
	}

	// Early exit si todos tienen la solución exacta
	allExact := true
	for _, client := range r.Clients {
		ans, ok := r.BestAnswers[client.ID]
		if !ok || ans.Distance != 0 {
			allExact = false
			break
		}
	}
	if allExact && len(r.Clients) > 0 {
		r.finishRound()
	}
}

func (r *GameRoom) handleSubmitLetras(c *Client, msg types.ClientMessage) {
	word := normalizeWord(strings.TrimSpace(msg.Word))
	if len([]rune(word)) < 5 || len([]rune(word)) > 10 {
		r.sendToast(c.ID, types.ToastMessage{Message: "La palabra debe tener entre 5 y 10 letras", Type: "error"})
		return
	}

	if !IsConstructible(word, r.State.Letters) {
		r.sendToast(c.ID, types.ToastMessage{Message: "Letras inválidas", Type: "error"})
		return
	}

	valid, orig := IsValidWord(word)
	if !valid {
		r.sendToast(c.ID, types.ToastMessage{Message: "La palabra no existe", Type: "error"})
		return
	}

	prev, hasPrev := r.BestAnswers[c.ID]
	if !hasPrev || len([]rune(orig)) >= len([]rune(prev.Word)) {
		r.BestAnswers[c.ID] = types.PlayerResult{
			PlayerID: c.ID,
			Name:     c.Player.Name,
			Word:     orig,
		}
		r.sendToast(c.ID, types.ToastMessage{
			Message: fmt.Sprintf("Palabra guardada: %s (%d letras)", orig, len([]rune(orig))),
			Type:    "success",
		})
		// Send structured word acceptance so the frontend doesn't need to parse toast text
		acceptMsg := types.ServerMessage{Type: "WORD_ACCEPTED", Payload: map[string]interface{}{"word": orig}}
		select {
		case c.SendChan <- acceptMsg:
		default:
		}
	}
}

func (r *GameRoom) finishRound() {
	r.State.State = types.StateFinished
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	r.State.EndTime = 0

	for _, c := range r.Clients {
		c.Player.IsReady = false
	}

	if r.State.RoundType == types.RoundCifras {
		r.finishCifras()
	} else {
		r.finishLetras()
	}

	r.broadcastState()
}

func (r *GameRoom) finishCifras() {
	var results []types.PlayerResult
	bestDiff := math.MaxInt

	for _, res := range r.BestAnswers {
		results = append(results, res)
		if res.Distance < bestDiff {
			bestDiff = res.Distance
		}
	}

	var winnerIDs []string
	winnerSet := make(map[string]bool)
	for i := range results {
		if results[i].Distance == bestDiff {
			if bestDiff == 0 {
				results[i].Points = 10
			} else {
				results[i].Points = 7
			}
			winnerIDs = append(winnerIDs, results[i].PlayerID)
			winnerSet[results[i].PlayerID] = true
			if c, ok := r.Clients[results[i].PlayerID]; ok {
				c.Player.Score += results[i].Points
			}
		}
	}
	r.State.OtherResults = results

	var winnerNames []string
	for _, res := range results {
		if winnerSet[res.PlayerID] {
			winnerNames = append(winnerNames, res.Name)
		}
	}
	r.State.Winner = determineWinner(winnerNames)
	if len(results) == 0 {
		r.State.Solution = "Nadie envió una respuesta a tiempo."
	}

	var found bool
	var expr string
	// Solo resolvemos si nadie encontró exacto o no hay ganadores
	if bestDiff > 0 || len(winnerIDs) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		found, expr = SolveCifras(ctx, r.State.Numbers, r.State.TargetNumber, 1_000_000)
		cancel()
	}
	
	if len(winnerIDs) > 0 {
		var winnerExpr string
		for _, res := range results {
			if res.PlayerID == winnerIDs[0] {
				winnerExpr = res.Expression
				break
			}
		}
		if winnerExpr != "" {
			r.State.SolutionSteps = strings.Split(winnerExpr, "\n")
		} else {
			r.State.SolutionSteps = nil
		}
	} else {
		r.State.SolutionSteps = nil
	}

	if found {
		r.State.ExactSolutionSteps = strings.Split(expr, "\n")
		r.State.Solution = "Solución exacta:"
	} else {
		r.State.ExactSolutionSteps = nil
		if len(winnerIDs) == 0 {
			r.State.Solution = "No se encontró solución exacta."
		}
	}
}

func (r *GameRoom) finishLetras() {
	var results []types.PlayerResult
	bestLen := 0

	for _, res := range r.BestAnswers {
		l := len([]rune(res.Word))
		if l > bestLen {
			bestLen = l
		}
	}
	for _, res := range r.BestAnswers {
		l := len([]rune(res.Word))
		if l == bestLen {
			res.Points = l
			if c, ok := r.Clients[res.PlayerID]; ok {
				c.Player.Score += l
			}
		} else {
			res.Points = 0
		}
		results = append(results, res)
	}
	r.State.OtherResults = results

	var winnerNames []string
	for _, res := range results {
		if len([]rune(res.Word)) == bestLen {
			winnerNames = append(winnerNames, res.Name)
		}
	}
	r.State.Winner = determineWinner(winnerNames)
	if len(results) == 0 {
		r.State.Solution = "Nadie envió una palabra a tiempo."
	}

	bestWords := GetBestWords(r.State.Letters)
	if len(bestWords) > 0 {
		r.State.Solution = fmt.Sprintf("Mejores palabras: %v", bestWords)
		r.State.ExactSolutionSteps = bestWords
	} else {
		r.State.Solution = "No se encontraron palabras."
		r.State.ExactSolutionSteps = nil
	}
	r.State.SolutionSteps = nil
}

func (r *GameRoom) resetToLobby() {
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	r.State = types.SyncData{State: types.StateLobby}
	r.BestAnswers = make(map[string]types.PlayerResult)
	// Limpiar jugadores históricos cuando la sala queda vacía
	r.HistoricalPlayers = make(map[string]types.Player)
}

func generateNumbers() []int {
	big := []int{25, 50, 75, 100}
	small := []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10}

	shuffleSlice(big)
	shuffleSlice(small)

	var res []int
	res = append(res, big[:2]...)
	res = append(res, small[:4]...)
	return res
}

func shuffleSlice[T any](s []T) {
	rand.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

func generateLetters(vowelsCount int) []string {
	vowelPool := []string{"A", "A", "A", "A", "E", "E", "E", "E", "I", "I", "I", "O", "O", "O", "U", "U"}
	consonantPool := []string{
		"B", "B", "C", "C", "C", "D", "D", "D", "F", "G", "G", "H", "H", "J",
		"L", "L", "L", "M", "M", "M", "N", "N", "N", "N", "Ñ", "P", "P", "Q",
		"R", "R", "R", "R", "S", "S", "S", "S", "T", "T", "T", "V", "X", "Y", "Z",
	}

	shuffleSlice(vowelPool)
	shuffleSlice(consonantPool)

	vowels := vowelPool[:min(vowelsCount, len(vowelPool))]
	consonants := consonantPool[:min(10-vowelsCount, len(consonantPool))]

	letters := make([]string, 0, 10)
	letters = append(letters, vowels...)
	letters = append(letters, consonants...)
	shuffleSlice(letters)
	return letters
}

func consumeNumber(available map[int]int, n int) bool {
	if n < 0 || available[n] <= 0 {
		return false
	}
	available[n]--
	return true
}

func computeOperation(a, b int, op string) (int, bool) {
	switch op {
	case "+":
		return a + b, true
	case "-":
		return a - b, true
	case "*":
		return a * b, true
	case "/":
		if b == 0 || a%b != 0 {
			return 0, false
		}
		return a / b, true
	default:
		return 0, false
	}
}

func determineWinner(names []string) string {
	switch len(names) {
	case 0:
		return "Nadie"
	case 1:
		return names[0]
	default:
		return "Empate"
	}
}

