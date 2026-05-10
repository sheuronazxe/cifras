package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

// ─────────────────────────────────────────────
// Constants
// ─────────────────────────────────────────────

const (
	StatusWaiting  = "waiting"
	StatusChoosing = "choosing"
	StatusPlaying  = "playing"
	StatusFinished = "finished"

	RoundCifras = "cifras"
	RoundLetras = "letras"
)

const (
	NumbersRoundDuration = 50 * time.Second
	LettersRoundDuration = 50 * time.Second
	GracePeriod          = 2 * time.Second
	ChooserTimeout       = 10 * time.Second
	ReadyTimeout         = 30 * time.Second
	MaxPlayerName        = 20
	MaxClients           = 20
	MinPlayersToStart    = 1
	ReadTimeout          = 60 * time.Second
	PingInterval         = 30 * time.Second
	PingTimeout          = 5 * time.Second
	SolverMaxSteps       = 5_000_000
	MaxSubmissionSteps   = 20
	AlphabetSize         = 27 // a-z + ñ
)

// ─────────────────────────────────────────────
// Data Models
// ─────────────────────────────────────────────

type Client struct {
	conn   *websocket.Conn
	mu     sync.Mutex
	name   string
	ready  bool
	sendCh chan Message
}

func (c *Client) send(msg Message) {
	select {
	case c.sendCh <- msg:
	default:
	}
}

type PlayerResult struct {
	Name        string `json:"name"`
	FinalNumber int    `json:"finalNumber,omitempty"`
	Distance    int    `json:"distance,omitempty"`
	Word        string `json:"word,omitempty"`
	Points      int    `json:"points"`
}

type GameState struct {
	RoundType          string         `json:"roundType"`
	Status             string         `json:"status"`
	Chooser            string         `json:"chooser,omitempty"`
	Target             int            `json:"target,omitempty"`
	Numbers            []int          `json:"numbers,omitempty"`
	Letters            []string       `json:"letters,omitempty"`
	Winner             string         `json:"winner"`
	Solution           string         `json:"solution"`
	SolutionSteps      []string       `json:"solutionSteps,omitempty"`
	ExactSolutionSteps []string       `json:"exactSolutionSteps,omitempty"`
	EndTime            int64          `json:"endTime"`
	ServerNow          int64          `json:"serverNow"`
	Rankings           map[string]int `json:"rankings"`
	OtherResults       []PlayerResult `json:"otherResults,omitempty"`
	TotalRounds        int            `json:"totalRounds"`
}

type Submission struct {
	Client      *Client
	Distance    int
	FinalNumber int
	Expression  string
	Word        string
	SubmitTime  time.Time
}

type DictEntry struct {
	original string
	word     string
	freq     [AlphabetSize]int
	length   int
}

type PlayerInfo struct {
	Name  string `json:"name"`
	Ready bool   `json:"ready"`
}

type Message struct {
	Type         string       `json:"type"`
	Name         string       `json:"name,omitempty"`
	Expression   string       `json:"expression,omitempty"`
	FinalNumber  int          `json:"finalNumber,omitempty"`
	Word         string       `json:"word,omitempty"`
	Vowels       int          `json:"vowels,omitempty"`
	State        GameState    `json:"state,omitempty"`
	Error        string       `json:"error,omitempty"`
	Info         string       `json:"info,omitempty"`
	Players      []PlayerInfo `json:"players,omitempty"`
}

// ─────────────────────────────────────────────
// Global State
// ─────────────────────────────────────────────

var (
	upgrader = websocket.Upgrader{
		CheckOrigin:     func(r *http.Request) bool { return true },
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	clients   = make(map[*Client]bool)
	clientsMu sync.RWMutex

	gameState = GameState{
		Status:   StatusWaiting,
		Rankings: make(map[string]int),
	}
	gameMu sync.RWMutex

	roundTimer       *time.Timer
	chooserTimer     *time.Timer
	readyTimer       *time.Timer
	timerMu          sync.Mutex
	roundStarting    bool
	roundStartMu     sync.Mutex
	roundSubmissions = make(map[*Client]Submission)
	submissionMu     sync.Mutex

	stepRegex = regexp.MustCompile(`^(\d+)\s*([+\-*/])\s*(\d+)\s*=\s*(\d+)$`)

	dictionary map[string]string
	sortedDict     []DictEntry
	dictByLength   [11][]int // length -> indices in sortedDict
	nextChooserIndex = 0
)

var normalizeReplacer = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u",
)

// ─────────────────────────────────────────────
// Dictionary Loading
// ─────────────────────────────────────────────

func loadDictionary() error {
	dictionary = make(map[string]string)
	f, err := os.Open("assets/diccionario.txt")
	if err != nil {
		return fmt.Errorf("abrir diccionario: %w", err)
	}
	defer f.Close()

	seen := make(map[string]struct{})
	var rawWords []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		w := strings.TrimSpace(scanner.Text())
		if w == "" {
			continue
		}
		original := w
		norm := normalizeWord(w)
		if _, exists := dictionary[norm]; !exists {
			dictionary[norm] = original
		}
		if _, ok := seen[norm]; !ok {
			seen[norm] = struct{}{}
			rawWords = append(rawWords, norm)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("leer diccionario: %w", err)
	}
	buildSortedDict(rawWords)
	log.Printf("Diccionario cargado con %d palabras (%d válidas para búsqueda)", len(dictionary), len(sortedDict))
	return nil
}

func buildSortedDict(rawWords []string) {
	sortedDict = make([]DictEntry, 0, len(rawWords))
	for _, word := range rawWords {
		runes := []rune(word)
		if len(runes) < 5 {
			continue
		}
		var freq [AlphabetSize]int
		valid := true
		for _, r := range runes {
			idx := runeIndex(r)
			if idx < 0 {
				valid = false
				break
			}
			freq[idx]++
		}
		if valid {
			original := dictionary[word]
			sortedDict = append(sortedDict, DictEntry{original, word, freq, len(runes)})
		}
	}
	sort.Slice(sortedDict, func(i, j int) bool {
		return sortedDict[i].length > sortedDict[j].length
	})

	// Build index by length
	for i := range dictByLength {
		dictByLength[i] = nil
	}
	for idx, entry := range sortedDict {
		if entry.length >= 5 && entry.length <= 10 {
			dictByLength[entry.length] = append(dictByLength[entry.length], idx)
		}
	}
}

var asciiIndex [128]int

func init() {
	for i := range asciiIndex {
		asciiIndex[i] = -1
	}
	for c := 'a'; c <= 'z'; c++ {
		asciiIndex[c] = int(c - 'a')
	}
}

func normalizeWord(w string) string {
	return normalizeReplacer.Replace(strings.ToLower(w))
}

func runeIndex(r rune) int {
	if r < 128 {
		return asciiIndex[r]
	}
	if r == 'ñ' {
		return 26
	}
	return -1
}

func letterFrequency(letters []string) [AlphabetSize]int {
	var freq [AlphabetSize]int
	for _, l := range letters {
		for _, r := range normalizeWord(l) {
			if idx := runeIndex(r); idx >= 0 {
				freq[idx]++
			}
		}
	}
	return freq
}

// ─────────────────────────────────────────────
// Entry Point
// ─────────────────────────────────────────────

func main() {
	if err := loadDictionary(); err != nil {
		log.Fatalf("Error cargando diccionario: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("./public")))
	mux.HandleFunc("/ws", handleConnections)
	mux.HandleFunc("/health", healthCheck)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("Servidor Cifras Multijugador iniciado en http://localhost%s\n", addr)
		log.Println("📡 Esperando conexiones...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error iniciando servidor: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("🛑 Apagando servidor...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

// ─────────────────────────────────────────────
// WebSocket Handler
// ─────────────────────────────────────────────

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error al actualizar WebSocket: %v", err)
		return
	}

	ws.SetReadLimit(4096)
	ws.SetReadDeadline(time.Now().Add(ReadTimeout))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(ReadTimeout))
		return nil
	})

	client := &Client{
		conn:   ws,
		name:   "Anónimo",
		sendCh: make(chan Message, 256),
	}

	// Atomic check-and-add under write lock to prevent TOCTOU race
	clientsMu.Lock()
	if len(clients) >= MaxClients {
		clientsMu.Unlock()
		ws.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseTryAgainLater, "Servidor lleno"))
		ws.Close()
		return
	}
	clients[client] = true
	totalClients := len(clients)
	clientsMu.Unlock()

	log.Printf("🔌 Cliente conectado desde %s (total: %d)", r.RemoteAddr, totalClients)

	gameMu.RLock()
	st := gameState
	st.ServerNow = time.Now().Unix()
	gameMu.RUnlock()

	client.send(Message{Type: "state", State: st})

	broadcastPlayers()
	go pingLoop(client)
	go writePump(client)
	handleMessages(client)
	cleanupClient(client)
}

func writePump(client *Client) {
	for msg := range client.sendCh {
		client.mu.Lock()
		err := client.conn.WriteJSON(msg)
		client.mu.Unlock()
		if err != nil {
			log.Printf("Error enviando a %s: %v", client.name, err)
			client.conn.Close()
			return
		}
	}
}

func pingLoop(client *Client) {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for range ticker.C {
		client.mu.Lock()
		err := client.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(PingTimeout))
		client.mu.Unlock()
		if err != nil {
			return
		}
	}
}

func handleMessages(client *Client) {
	for {
		var msg Message
		if err := client.conn.ReadJSON(&msg); err != nil {
			return
		}

		switch msg.Type {
		case "join":
			handleJoin(client, msg)
		case "ready":
			handleReady(client)
		case "submit":
			handleSubmission(client, msg)
		case "vowels":
			handleVowels(client, msg.Vowels)
		default:
			client.send(Message{Type: "error", Error: "Tipo de mensaje desconocido"})
		}
	}
}

func cleanupClient(client *Client) {
	client.conn.Close()
	close(client.sendCh) // unblocks writePump goroutine

	clientsMu.Lock()
	delete(clients, client)
	remainingClients := len(clients)
	clientsMu.Unlock()

	log.Printf("🔌 Cliente '%s' desconectado (total: %d)", client.name, remainingClients)

	submissionMu.Lock()
	delete(roundSubmissions, client)
	submissionMu.Unlock()

	broadcastPlayers()

	gameMu.Lock()
	defer gameMu.Unlock()

	switch gameState.Status {
	case StatusChoosing:
		if remainingClients == 0 {
			resetGameState()
			return
		}
		if client.name == gameState.Chooser {
			clientsMu.RLock()
			var newChooser *Client
			for c := range clients {
				newChooser = c
				break
			}
			clientsMu.RUnlock()
			if newChooser != nil {
				gameState.Chooser = newChooser.name
				st := gameState
				st.ServerNow = time.Now().Unix()
				broadcast(Message{Type: "state", State: st})
			}
		}
	case StatusPlaying:
		if remainingClients == 0 {
			timerMu.Lock()
			if roundTimer != nil {
				roundTimer.Stop()
				roundTimer = nil
			}
			timerMu.Unlock()
			resetGameState()
		}
	case StatusFinished:
		if remainingClients == 0 {
			resetGameState()
			return
		}
		go checkAllReady()
	case StatusWaiting:
		go checkAllReady()
	}
}

func resetGameState() {
	gameState.Status = StatusWaiting
	gameState.RoundType = ""
	gameState.Chooser = ""
	gameState.Numbers = nil
	gameState.Letters = nil
	gameState.Solution = ""
	gameState.Winner = ""
	gameState.SolutionSteps = nil
	gameState.ExactSolutionSteps = nil
	gameState.EndTime = 0
	gameState.TotalRounds = 0
	gameState.Rankings = make(map[string]int)
}

// ─────────────────────────────────────────────
// Player Management
// ─────────────────────────────────────────────

func handleJoin(client *Client, msg Message) {
	name := sanitizeName(strings.TrimSpace(msg.Name))
	if name == "" {
		name = "Anónimo"
	}
	if len(name) > MaxPlayerName {
		name = name[:MaxPlayerName]
	}

	clientsMu.Lock()
	existing := make(map[string]struct{}, len(clients))
	for c := range clients {
		if c != client {
			existing[c.name] = struct{}{}
		}
	}
	baseName := name
	for i := 1; ; i++ {
		if _, ok := existing[name]; !ok {
			break
		}
		name = fmt.Sprintf("%s#%d", baseName, i)
	}
	client.name = name
	clientsMu.Unlock()

	log.Printf("👤 Jugador '%s' se unió", name)
	broadcastPlayers()
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

func broadcastPlayers() {
	clientsMu.RLock()
	players := make([]PlayerInfo, 0, len(clients))
	for c := range clients {
		players = append(players, PlayerInfo{Name: c.name, Ready: c.ready})
	}
	clientsMu.RUnlock()

	broadcast(Message{Type: "players", Players: players})
}

func handleReady(client *Client) {
	gameMu.RLock()
	status := gameState.Status
	gameMu.RUnlock()

	if status != StatusWaiting && status != StatusFinished {
		return
	}

	clientsMu.Lock()
	client.ready = true
	clientsMu.Unlock()

	log.Printf("✅ Jugador '%s' está listo (estado: %s)", client.name, status)
	broadcastPlayers()
	checkAllReady()
}

func checkAllReady() {
	clientsMu.RLock()
	count := len(clients)
	allReady := true
	for c := range clients {
		if !c.ready {
			allReady = false
			break
		}
	}
	clientsMu.RUnlock()

	if count < MinPlayersToStart {
		return
	}

	if allReady {
		timerMu.Lock()
		if readyTimer != nil {
			readyTimer.Stop()
			readyTimer = nil
		}
		timerMu.Unlock()
		go startNewRound()
		return
	}

	timerMu.Lock()
	if readyTimer == nil {
		readyTimer = time.AfterFunc(ReadyTimeout, func() {
			clientsMu.RLock()
			currentCount := len(clients)
			clientsMu.RUnlock()
			if currentCount >= MinPlayersToStart {
				go startNewRound()
			}
		})
	}
	timerMu.Unlock()
}

// ─────────────────────────────────────────────
// Round Lifecycle
// ─────────────────────────────────────────────

func startNewRound() {
	roundStartMu.Lock()
	if roundStarting {
		roundStartMu.Unlock()
		return
	}
	roundStarting = true
	roundStartMu.Unlock()

	defer func() {
		roundStartMu.Lock()
		roundStarting = false
		roundStartMu.Unlock()
	}()

	timerMu.Lock()
	if readyTimer != nil {
		readyTimer.Stop()
		readyTimer = nil
	}
	timerMu.Unlock()

	gameMu.Lock()
	if gameState.Status != StatusWaiting && gameState.Status != StatusFinished {
		gameMu.Unlock()
		return
	}
	gameMu.Unlock()

	clientsMu.Lock()
	var players []*Client
	for c := range clients {
		c.ready = false
		players = append(players, c)
	}
	clientsMu.Unlock()
	broadcastPlayers()

	submissionMu.Lock()
	roundSubmissions = make(map[*Client]Submission)
	submissionMu.Unlock()

	gameMu.Lock()
	timerMu.Lock()
	if roundTimer != nil {
		roundTimer.Stop()
	}
	timerMu.Unlock()

	gameState.TotalRounds++
	gameState.Winner = ""
	gameState.Solution = ""
	gameState.SolutionSteps = nil
	gameState.ExactSolutionSteps = nil
	gameState.OtherResults = nil

	if gameState.TotalRounds%2 == 1 {
		gameState.RoundType = RoundCifras
		gameState.Status = StatusPlaying
		gameState.Numbers = generateNumbers()
		gameState.Target = rand.IntN(899) + 101
		gameState.Letters = nil

		now := time.Now()
		gameState.EndTime = now.Add(NumbersRoundDuration).Unix()
		gameState.ServerNow = now.Unix()

		timerMu.Lock()
		roundTimer = time.AfterFunc(NumbersRoundDuration, timeOutRound)
		timerMu.Unlock()

		log.Printf("🎮 Ronda #%d CIFRAS iniciada - Objetivo: %d - Números: %v", gameState.TotalRounds, gameState.Target, gameState.Numbers)
	} else {
		gameState.RoundType = RoundLetras
		gameState.Status = StatusChoosing
		gameState.Numbers = nil
		gameState.Target = 0

		sort.Slice(players, func(i, j int) bool { return players[i].name < players[j].name })
		chooser := players[nextChooserIndex%len(players)]
		nextChooserIndex++
		gameState.Chooser = chooser.name
		gameState.EndTime = 0
		gameState.ServerNow = time.Now().Unix()

		timerMu.Lock()
		if chooserTimer != nil {
			chooserTimer.Stop()
		}
		chooserTimer = time.AfterFunc(ChooserTimeout, timeoutChooser)
		timerMu.Unlock()

		log.Printf("🎮 Ronda #%d LETRAS iniciada - Chooser: %s (%d jugadores)", gameState.TotalRounds, chooser.name, len(players))
	}

	st := gameState
	gameMu.Unlock()

	broadcast(Message{Type: "state", State: st})
}

func handleVowels(client *Client, count int) {
	gameMu.Lock()
	defer gameMu.Unlock()

	if gameState.Status != StatusChoosing || gameState.RoundType != RoundLetras {
		return
	}
	if client.name != gameState.Chooser {
		return
	}

	count = max(3, min(5, count))
	startLettersRound(count)
}

func timeoutChooser() {
	gameMu.Lock()
	defer gameMu.Unlock()

	if gameState.Status != StatusChoosing || gameState.RoundType != RoundLetras {
		return
	}

	vowelCount := rand.IntN(3) + 3
	startLettersRound(vowelCount)
}

// startLettersRound begins the letters playing phase. Caller must hold gameMu.Lock.
func startLettersRound(vowelCount int) {
	vowels := getVowels(vowelCount)
	consonants := getConsonants(10 - vowelCount)

	// Create a new slice to avoid aliasing the backing arrays of vowels/consonants
	letters := make([]string, 0, 10)
	letters = append(letters, vowels...)
	letters = append(letters, consonants...)
	shuffleSlice(letters)

	gameState.Letters = letters
	gameState.Status = StatusPlaying
	now := time.Now()
	gameState.EndTime = now.Add(LettersRoundDuration).Unix()
	gameState.ServerNow = now.Unix()

	timerMu.Lock()
	if chooserTimer != nil {
		chooserTimer.Stop()
		chooserTimer = nil
	}
	if roundTimer != nil {
		roundTimer.Stop()
	}
	roundTimer = time.AfterFunc(LettersRoundDuration, timeOutRound)
	timerMu.Unlock()

	st := gameState
	broadcast(Message{Type: "state", State: st})

	log.Printf("🎮 Ronda #%d LETRAS - Letras: %v", gameState.TotalRounds, letters)
}

func shuffleSlice[T any](s []T) {
	rand.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

func getVowels(count int) []string {
	pool := []string{"A", "A", "A", "A", "E", "E", "E", "E", "I", "I", "I", "O", "O", "O", "U", "U"}
	shuffleSlice(pool)
	return pool[:min(count, len(pool))]
}

func getConsonants(count int) []string {
	pool := []string{
		"B", "B", "C", "C", "C", "D", "D", "D", "F", "G", "G", "H", "H", "J",
		"L", "L", "L", "M", "M", "M", "N", "N", "N", "N", "Ñ", "P", "P", "Q",
		"R", "R", "R", "R", "S", "S", "S", "S", "T", "T", "T", "V", "X", "Y", "Z",
	}
	shuffleSlice(pool)
	return pool[:min(count, len(pool))]
}

func generateNumbers() []int {
	smallPool := []int{1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10}
	largePool := []int{25, 50, 75, 100}

	shuffleSlice(smallPool)
	shuffleSlice(largePool)

	return append(largePool[:2], smallPool[:4]...)
}

func timeOutRound() {
	gameMu.RLock()
	if gameState.Status != StatusPlaying {
		gameMu.RUnlock()
		return
	}
	gameMu.RUnlock()

	time.AfterFunc(GracePeriod, func() {
		finishRound()
	})
}

func finishRound() {
	gameMu.Lock()
	if gameState.Status != StatusPlaying {
		gameMu.Unlock()
		return
	}
	gameState.Status = StatusFinished
	gameMu.Unlock()

	clientsMu.Lock()
	for c := range clients {
		c.ready = false
	}
	clientsMu.Unlock()
	broadcastPlayers()

	// Copy submissions under lock, then release before acquiring gameMu.
	// This avoids holding submissionMu while locking gameMu.
	submissionMu.Lock()
	subs := make(map[*Client]Submission, len(roundSubmissions))
	for k, v := range roundSubmissions {
		subs[k] = v
	}
	submissionMu.Unlock()

	gameMu.Lock()
	if gameState.RoundType == RoundCifras {
		finishCifrasRound(subs)
	} else {
		finishLetrasRound(subs)
	}
	st := gameState
	gameMu.Unlock()

	broadcast(Message{Type: "state", State: st})
}

// finishCifrasRound processes the cifras round results. Caller must hold gameMu.Lock.
func finishCifrasRound(subs map[*Client]Submission) {
	if len(subs) == 0 {
		gameState.Winner = "Nadie"
		gameState.Solution = "Nadie envió una respuesta a tiempo."
		gameState.SolutionSteps = nil
		gameState.ExactSolutionSteps = findExactSolution(gameState.Numbers, gameState.Target)
		gameState.OtherResults = nil
		return
	}

	bestDist := math.MaxInt
	for _, sub := range subs {
		if sub.Distance < bestDist {
			bestDist = sub.Distance
		}
	}

	var winners []*Client
	for c, sub := range subs {
		if sub.Distance == bestDist {
			winners = append(winners, c)
		}
	}

	pts := 7
	if bestDist == 0 {
		pts = 10
	}

	var winnerNames []string
	for _, w := range winners {
		winnerNames = append(winnerNames, w.name)
		gameState.Rankings[w.name] += pts
	}

	if len(winners) > 1 {
		gameState.Winner = "Empate"
		gameState.Solution = fmt.Sprintf("Empate a %d (distancia %d) entre: %s - %d pts", subs[winners[0]].FinalNumber, bestDist, strings.Join(winnerNames, ", "), pts)
	} else {
		gameState.Winner = winners[0].name
		gameState.Solution = fmt.Sprintf("Logró %d (a %d del objetivo) - %d puntos", subs[winners[0]].FinalNumber, bestDist, pts)
	}

	if bestDist == 0 {
		var firstExact *Client
		var firstTime time.Time
		for c, sub := range subs {
			if sub.Distance == 0 && (firstExact == nil || sub.SubmitTime.Before(firstTime)) {
				firstExact = c
				firstTime = sub.SubmitTime
			}
		}
		gameState.SolutionSteps = splitSteps(subs[firstExact].Expression)
		gameState.ExactSolutionSteps = nil
	} else {
		exactSteps := findExactSolution(gameState.Numbers, gameState.Target)
		if exactSteps != nil {
			gameState.SolutionSteps = nil
			gameState.ExactSolutionSteps = exactSteps
		} else {
			gameState.SolutionSteps = splitSteps(subs[winners[0]].Expression)
			gameState.ExactSolutionSteps = nil
		}
	}

	var others []PlayerResult
	for _, sub := range subs {
		others = append(others, PlayerResult{
			Name:        sub.Client.name,
			FinalNumber: sub.FinalNumber,
			Distance:    sub.Distance,
		})
	}
	gameState.OtherResults = others
}

// finishLetrasRound processes the letras round results. Caller must hold gameMu.Lock.
func finishLetrasRound(subs map[*Client]Submission) {
	gameState.ExactSolutionSteps = findBestWords(gameState.Letters)

	if len(subs) == 0 {
		gameState.Winner = "Nadie"
		gameState.Solution = "Nadie envió una palabra a tiempo."
		gameState.OtherResults = nil
		return
	}

	maxLength := 0
	for _, sub := range subs {
		if wordLen := utf8.RuneCountInString(sub.Word); wordLen > maxLength {
			maxLength = wordLen
		}
	}

	var winners []string
	var winnerDetails []string
	var singleWinnerWord string
	var others []PlayerResult

	for c, sub := range subs {
		wordLen := utf8.RuneCountInString(sub.Word)
		others = append(others, PlayerResult{
			Name:   c.name,
			Word:   strings.ToUpper(sub.Word),
			Points: wordLen,
		})
		if wordLen == maxLength {
			winners = append(winners, c.name)
			winnerDetails = append(winnerDetails, fmt.Sprintf("%s (%s)", c.name, strings.ToUpper(sub.Word)))
			singleWinnerWord = strings.ToUpper(sub.Word)
			gameState.Rankings[c.name] += maxLength
		}
	}

	if len(winners) > 1 {
		gameState.Winner = "Empate"
		gameState.Solution = fmt.Sprintf("Empate a %d letras entre: %s", maxLength, strings.Join(winnerDetails, ", "))
	} else {
		gameState.Winner = winners[0]
		gameState.Solution = fmt.Sprintf("Mejor palabra: %s (%d puntos)", singleWinnerWord, maxLength)
	}

	gameState.OtherResults = others
}

func findBestWords(letters []string) []string {
	available := letterFrequency(letters)

	var result []string
	minLength := -1

	// Iterate from longest to shortest using the index
	for length := 10; length >= 5; length-- {
		if len(result) >= 5 && length < minLength {
			break
		}

		for _, idx := range dictByLength[length] {
			entry := sortedDict[idx]
			if len(result) >= 5 && entry.length < minLength {
				break
			}

			valid := true
			for i := 0; i < AlphabetSize; i++ {
				if entry.freq[i] > available[i] {
					valid = false
					break
				}
			}
			if valid {
				result = append(result, entry.original)
				minLength = entry.length
			}
		}
	}

	if len(result) > 5 {
		var top, tied []string
		for _, w := range result {
			if utf8.RuneCountInString(w) > minLength {
				top = append(top, w)
			} else {
				tied = append(tied, w)
			}
		}
		needed := 5 - len(top)
		if needed > 0 {
			rand.Shuffle(len(tied), func(i, j int) { tied[i], tied[j] = tied[j], tied[i] })
			result = append(top, tied[:needed]...)
		} else {
			result = top
		}
	}

	for i, w := range result {
		result[i] = strings.ToUpper(w)
	}
	return result
}

func splitSteps(expr string) []string {
	var steps []string
	for _, s := range strings.Split(expr, ";") {
		s = strings.TrimSpace(s)
		if s != "" {
			steps = append(steps, s)
		}
	}
	return steps
}

// ─────────────────────────────────────────────
// Exact Solution Solver
// ─────────────────────────────────────────────

type solveExpr struct {
	Value int
	Steps []string
}

func findExactSolution(numbers []int, target int) []string {
	exprs := make([]solveExpr, len(numbers))
	for i, n := range numbers {
		exprs[i] = solveExpr{Value: n}
	}

	steps := 0
	var solve func([]solveExpr) []string
	solve = func(current []solveExpr) []string {
		steps++
		if steps > SolverMaxSteps {
			return nil
		}

		for _, e := range current {
			if e.Value == target {
				return e.Steps
			}
		}
		if len(current) <= 1 {
			return nil
		}

		for i := 0; i < len(current); i++ {
			for j := i + 1; j < len(current); j++ {
				e1, e2 := current[i], current[j]
				for _, op := range []string{"+", "-", "*", "/"} {
					a, b := e1.Value, e2.Value

					// Normalize: ensure a >= b for subtraction/division
					if op == "-" || op == "/" {
						if a < b {
							a, b = b, a
						}
					}
					// Skip identity operations that don't produce new values
					if (op == "+" || op == "-") && b == 0 {
						continue
					}
					if (op == "*" || op == "/") && b == 1 {
						continue
					}
					if op == "-" && a == b {
						continue
					}
					if op == "/" && (b == 0 || a%b != 0) {
						continue
					}
					// Skip redundant commutative ops when a == b
					if (op == "+" || op == "*") && a == b {
						continue
					}
					// Normalize commutative ops to canonical form (a >= b)
					if (op == "+" || op == "*") && a < b {
						a, b = b, a
					}

					var val int
					switch op {
					case "+":
						val = a + b
					case "-":
						val = a - b
					case "*":
						val = a * b
					case "/":
						val = a / b
					}

					if val <= 0 {
						continue
					}

					// Early exit if we found the target
					if val == target {
						stepStr := strconv.Itoa(a) + " " + op + " " + strconv.Itoa(b) + " = " + strconv.Itoa(val)
						result := make([]string, 0, len(e1.Steps)+len(e2.Steps)+1)
						result = append(result, e1.Steps...)
						result = append(result, e2.Steps...)
						result = append(result, stepStr)
						return result
					}

					stepStr := strconv.Itoa(a) + " " + op + " " + strconv.Itoa(b) + " = " + strconv.Itoa(val)
					newSteps := make([]string, 0, len(e1.Steps)+len(e2.Steps)+1)
					newSteps = append(newSteps, e1.Steps...)
					newSteps = append(newSteps, e2.Steps...)
					newSteps = append(newSteps, stepStr)

					newCurrent := make([]solveExpr, 0, len(current)-1)
					for k := 0; k < len(current); k++ {
						if k != i && k != j {
							newCurrent = append(newCurrent, current[k])
						}
					}
					newCurrent = append(newCurrent, solveExpr{Value: val, Steps: newSteps})

					if res := solve(newCurrent); res != nil {
						return res
					}
				}
			}
		}
		return nil
	}

	return solve(exprs)
}

// ─────────────────────────────────────────────
// Submission Validation
// ─────────────────────────────────────────────

func handleSubmission(client *Client, msg Message) {
	gameMu.RLock()
	status := gameState.Status
	rtype := gameState.RoundType
	gameMu.RUnlock()

	if status != StatusPlaying {
		return
	}

	if rtype == RoundCifras {
		handleCifrasSubmission(client, msg.Expression, msg.FinalNumber)
	} else {
		handleLetrasSubmission(client, msg.Word)
	}
}

func handleCifrasSubmission(client *Client, expression string, finalNumber int) {
	if expression == "" || len(expression) > 500 {
		return
	}

	steps := strings.Split(expression, ";")
	if len(steps) > MaxSubmissionSteps {
		return
	}

	gameMu.RLock()
	numbersCopy := make([]int, len(gameState.Numbers))
	copy(numbersCopy, gameState.Numbers)
	target := gameState.Target
	gameMu.RUnlock()

	available := make(map[int]int)
	for _, n := range numbersCopy {
		available[n]++
	}

	for _, step := range steps {
		step = strings.TrimSpace(step)
		if step == "" {
			continue
		}

		matches := stepRegex.FindStringSubmatch(step)
		if matches == nil {
			return
		}

		a, _ := strconv.Atoi(matches[1])
		b, _ := strconv.Atoi(matches[3])
		result, _ := strconv.Atoi(matches[4])
		op := matches[2]

		if !consumeNumber(available, a) || !consumeNumber(available, b) {
			return
		}

		expected, ok := computeOperation(a, b, op)
		if !ok || expected <= 0 || expected != result {
			return
		}
		available[result]++
	}

	if available[finalNumber] <= 0 || finalNumber < 1 || finalNumber > 9999 {
		return
	}

	dist := abs(finalNumber - target)
	now := time.Now()

	submissionMu.Lock()
	existing, exists := roundSubmissions[client]
	isBetter := !exists || dist < existing.Distance
	if isBetter {
		roundSubmissions[client] = Submission{
			Client:      client,
			Distance:    dist,
			FinalNumber: finalNumber,
			Expression:  expression,
			SubmitTime:  now,
		}
	}
	submissionMu.Unlock()

	if isBetter {
		client.send(Message{
			Type:        "accepted",
			FinalNumber: finalNumber,
			Info:        fmt.Sprintf("✔ Mejor respuesta: %d (distancia: %d)", finalNumber, dist),
		})
	}

	submissionMu.Lock()
	exactCount := 0
	for _, sub := range roundSubmissions {
		if sub.Distance == 0 {
			exactCount++
		}
	}
	submissionMu.Unlock()

	clientsMu.RLock()
	totalPlayers := len(clients)
	clientsMu.RUnlock()

	if exactCount > 0 && exactCount == totalPlayers {
		timerMu.Lock()
		if roundTimer != nil {
			roundTimer.Stop()
		}
		timerMu.Unlock()
		finishRound()
	}
}

func handleLetrasSubmission(client *Client, word string) {
	word = strings.TrimSpace(word)
	word = normalizeWord(word)
	if utf8.RuneCountInString(word) < 5 || utf8.RuneCountInString(word) > 10 {
		return
	}

	gameMu.RLock()
	lettersCopy := make([]string, len(gameState.Letters))
	copy(lettersCopy, gameState.Letters)
	gameMu.RUnlock()

	available := letterFrequency(lettersCopy)

	validLetters := true
	for _, r := range word {
		idx := runeIndex(r)
		if idx >= 0 && available[idx] > 0 {
			available[idx]--
		} else {
			validLetters = false
			break
		}
	}

	if !validLetters {
		client.send(Message{Type: "error", Error: "La palabra usa letras no disponibles."})
		return
	}

	originalWord, ok := dictionary[word]
	if !ok {
		client.send(Message{Type: "error", Error: "La palabra no está en el diccionario."})
		return
	}

	now := time.Now()
	submissionMu.Lock()
	existing, exists := roundSubmissions[client]
	isBetter := !exists || utf8.RuneCountInString(originalWord) > utf8.RuneCountInString(existing.Word)
	if isBetter {
		roundSubmissions[client] = Submission{
			Client:     client,
			Word:       originalWord,
			SubmitTime: now,
		}
	}
	submissionMu.Unlock()

	if isBetter {
		client.send(Message{
			Type: "accepted",
			Word: strings.ToUpper(originalWord),
			Info: fmt.Sprintf("✔ Palabra enviada: %s (%d puntos)", strings.ToUpper(originalWord), utf8.RuneCountInString(originalWord)),
		})
	}
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

// ─────────────────────────────────────────────
// Networking Helpers
// ─────────────────────────────────────────────

func broadcast(msg Message) {
	clientsMu.RLock()
	defer clientsMu.RUnlock()

	for c := range clients {
		select {
		case c.sendCh <- msg:
		default:
		}
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	clientsMu.RLock()
	count := len(clients)
	clientsMu.RUnlock()

	gameMu.RLock()
	status := gameState.Status
	gameMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"players": count,
		"game":    status,
	})
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
