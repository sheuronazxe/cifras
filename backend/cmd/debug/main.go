package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"cifras/internal/types"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Active client tracking
var (
	activeConn *websocket.Conn
	connMutex  sync.Mutex
	localState = 1 // 1: Lobby, 2: Choosing Letras, 3: Playing Letras, etc.
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/ws", serveWs)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Fallback to serve static files if requested
	mux.Handle("/", http.FileServer(http.Dir("../../frontend/build")))

	port := "8080"
	srv := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		log.Printf("\n=======================================================\n")
		log.Printf("🎮 SERVIDOR DE DEPURACIÓN (CIFRAS Y LETRAS) INICIADO 🎮\n")
		log.Printf("Escuchando en http://localhost:%s\n", port)
		log.Printf("Espera de conexión WebSocket de cliente en /ws...\n")
		log.Printf("=======================================================\n")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error del servidor: %v", err)
		}
	}()

	// Start terminal state switcher in a separate goroutine
	go terminalSwitcher()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Apagando servidor de depuración...")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Println("Error accepting websocket:", err)
		return
	}

	connMutex.Lock()
	if activeConn != nil {
		// Close previous connection if any
		activeConn.Close(websocket.StatusNormalClosure, "newer connection registered")
	}
	activeConn = conn
	connMutex.Unlock()

	log.Println("\n🔌 ¡Cliente conectado al WebSocket de depuración!")

	// Send WELCOME message immediately
	welcome := types.ServerMessage{
		Type: "WELCOME",
		Payload: types.Player{
			ID:      "debug-id",
			Name:    "Tú",
			Score:   0,
			IsReady: false,
		},
	}
	ctx := context.Background()
	if err := wsjson.Write(ctx, conn, welcome); err != nil {
		log.Println("Error writing WELCOME message:", err)
		return
	}

	// Broadcast the current state to the newly connected client
	sendCurrentState()

	// Read loop to print incoming client messages (useful to see actions)
	for {
		var msg types.ClientMessage
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure && websocket.CloseStatus(err) != websocket.StatusGoingAway {
				log.Printf("Error de lectura del cliente: %v", err)
			}
			break
		}
		log.Printf("\n📥 Mensaje recibido del cliente: Tipo=%s, Payload=%+v\n", msg.Type, msg)
	}

	connMutex.Lock()
	if activeConn == conn {
		activeConn = nil
	}
	connMutex.Unlock()
	log.Println("🔌 Cliente desconectado del WebSocket.")
}

func terminalSwitcher() {
	// Simple instructions
	time.Sleep(500 * time.Millisecond)
	printMenu()

	buf := make([]byte, 1024)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			log.Println("Error leyendo de consola:", err)
			continue
		}

		input := strings.TrimSpace(string(buf[:n]))

		if input == "" {
			// Advance sequentially
			localState++
			if localState > 7 {
				localState = 1
			}
		} else {
			// Jump to state
			val, err := strconv.Atoi(input)
			if err == nil && val >= 1 && val <= 7 {
				localState = val
			} else if strings.ToLower(input) == "menu" || strings.ToLower(input) == "h" {
				printMenu()
				continue
			} else {
				fmt.Println("⚠️  Entrada inválida. Pulsa ENTER para avanzar, escribe un número (1-7) para saltar, o 'menu' para ayuda.")
				continue
			}
		}

		sendCurrentState()
	}
}

func printMenu() {
	fmt.Printf("\n✨ CONTROLES DE LA SALA DE DEPURACIÓN DE DISEÑO ✨\n")
	fmt.Println("----------------------------------------------------------------------")
	fmt.Println(" Pulsa [ENTER] en esta terminal para AVANZAR secuencialmente de pantalla.")
	fmt.Println(" O escribe el número de la pantalla y pulsa [ENTER] para ir directo:")
	fmt.Println("   [1] LOBBY (Sala de espera)")
	fmt.Println("   [2] CHOOSING Letras (Fase de selección - Tú eliges)")
	fmt.Println("   [3] PLAYING Letras (Juego de letras activo - 10 letras)")
	fmt.Println("   [4] RESULTS Letras (Resultados de la ronda de letras - Tú ganas)")
	fmt.Println("   [5] CHOOSING Cifras (Fase de selección - María elige)")
	fmt.Println("   [6] PLAYING Cifras (Juego de cifras activo - Conseguir 524)")
	fmt.Println("   [7] RESULTS Cifras (Resultados de la ronda de cifras - Exacto por María)")
	fmt.Println(" Escribe 'menu' para volver a mostrar esta ayuda.")
	fmt.Printf("----------------------------------------------------------------------\n\n")
}

func sendCurrentState() {
	connMutex.Lock()
	conn := activeConn
	connMutex.Unlock()

	if conn == nil {
		fmt.Printf("\r[Estado %d] ⚠️  No hay ningún cliente conectado al WebSocket para recibir el estado.", localState)
		return
	}

	var syncData types.SyncData
	nowMs := time.Now().UnixNano() / int64(time.Millisecond)

	switch localState {
	case 1:
		fmt.Printf("\r🚀 Mostrando Pantalla: 1. LOBBY (Sala de espera)                     \n")
		syncData = types.SyncData{
			State:        types.StateLobby,
			CurrentRound: 0,
			Players: []types.Player{
				{ID: "debug-id", Name: "Tú", Score: 0, IsReady: false},
				{ID: "bot-1", Name: "María", Score: 0, IsReady: true},
				{ID: "bot-2", Name: "Carlos", Score: 0, IsReady: false},
			},
		}

	case 2:
		fmt.Printf("\r🚀 Mostrando Pantalla: 2. CHOOSING Letras (Fase selección - Tú)         \n")
		syncData = types.SyncData{
			State:        types.StateChoosing,
			RoundType:    types.RoundLetras,
			CurrentRound: 1,
			Chooser:      "Tú",
			ChooserID:    "debug-id",
			Players: []types.Player{
				{ID: "debug-id", Name: "Tú", Score: 0, IsReady: false},
				{ID: "bot-1", Name: "María", Score: 0, IsReady: false},
				{ID: "bot-2", Name: "Carlos", Score: 0, IsReady: false},
			},
		}

	case 3:
		fmt.Printf("\r🚀 Mostrando Pantalla: 3. PLAYING Letras (Juego de letras activo)       \n")
		syncData = types.SyncData{
			State:        types.StatePlaying,
			RoundType:    types.RoundLetras,
			CurrentRound: 1,
			Chooser:      "Tú",
			ChooserID:    "debug-id",
			Letters:      []string{"A", "B", "E", "R", "T", "U", "N", "D", "O", "S"},
			EndTime:      nowMs + 30000,
			Players: []types.Player{
				{ID: "debug-id", Name: "Tú", Score: 0, IsReady: false},
				{ID: "bot-1", Name: "María", Score: 0, IsReady: false},
				{ID: "bot-2", Name: "Carlos", Score: 0, IsReady: false},
			},
		}

	case 4:
		fmt.Printf("\r🚀 Mostrando Pantalla: 4. RESULTS Letras (Resultados de Letras)         \n")
		syncData = types.SyncData{
			State:        types.StateFinished,
			RoundType:    types.RoundLetras,
			CurrentRound: 1,
			Winner:       "Tú",
			Players: []types.Player{
				{ID: "debug-id", Name: "Tú", Score: 9, IsReady: false},
				{ID: "bot-1", Name: "María", Score: 0, IsReady: false},
				{ID: "bot-2", Name: "Carlos", Score: 0, IsReady: false},
			},
			OtherResults: []types.PlayerResult{
				{Name: "Tú", Word: "ABERTURA", Points: 9},
				{Name: "María", Word: "NO", Points: 0},
				{Name: "Carlos", Word: "SABER", Points: 5},
			},
		}

	case 5:
		fmt.Printf("\r🚀 Mostrando Pantalla: 5. CHOOSING Cifras (Fase selección - María)      \n")
		syncData = types.SyncData{
			State:        types.StateChoosing,
			RoundType:    types.RoundCifras,
			CurrentRound: 2,
			Chooser:      "María",
			ChooserID:    "bot-1",
			Players: []types.Player{
				{ID: "debug-id", Name: "Tú", Score: 9, IsReady: false},
				{ID: "bot-1", Name: "María", Score: 0, IsReady: false},
				{ID: "bot-2", Name: "Carlos", Score: 0, IsReady: false},
			},
		}

	case 6:
		fmt.Printf("\r🚀 Mostrando Pantalla: 6. PLAYING Cifras (Juego de cifras activo)       \n")
		syncData = types.SyncData{
			State:        types.StatePlaying,
			RoundType:    types.RoundCifras,
			CurrentRound: 2,
			Chooser:      "María",
			ChooserID:    "bot-1",
			TargetNumber: 524,
			Numbers:      []int{100, 4, 9, 2, 8, 25},
			EndTime:      nowMs + 45000,
			Players: []types.Player{
				{ID: "debug-id", Name: "Tú", Score: 9, IsReady: false},
				{ID: "bot-1", Name: "María", Score: 0, IsReady: false},
				{ID: "bot-2", Name: "Carlos", Score: 0, IsReady: false},
			},
		}

	case 7:
		fmt.Printf("\r🚀 Mostrando Pantalla: 7. RESULTS Cifras (Resultados de Cifras)         \n")
		syncData = types.SyncData{
			State:        types.StateFinished,
			RoundType:    types.RoundCifras,
			CurrentRound: 2,
			Winner:       "María",
			Players: []types.Player{
				{ID: "debug-id", Name: "Tú", Score: 15, IsReady: false},
				{ID: "bot-1", Name: "María", Score: 10, IsReady: false},
				{ID: "bot-2", Name: "Carlos", Score: 5, IsReady: false},
			},
			OtherResults: []types.PlayerResult{
				{Name: "Tú", FinalNumber: 520, Distance: 4, Expression: "(100 * 5) + 25 - 9 + 4", Points: 6},
				{Name: "María", FinalNumber: 524, Distance: 0, Expression: "(100 * 5) + 25 - 9 + 8", Points: 10},
				{Name: "Carlos", FinalNumber: 516, Distance: 8, Expression: "(100 * 5) + 25 - 9", Points: 5},
			},
			ExactSolutionSteps: []string{
				"100 * 5 = 500",
				"500 + 25 = 525",
				"525 - 9 = 516",
				"516 + 8 = 524",
			},
		}
	}

	syncData.ServerTime = time.Now().UnixMilli()

	msg := types.ServerMessage{
		Type:    "SYNC",
		Payload: syncData,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, conn, msg); err != nil {
		log.Println("Error enviando estado al cliente:", err)
	}
}
