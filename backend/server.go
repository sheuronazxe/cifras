package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cifras/internal/types"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const (
	PingInterval      = 30 * time.Second
	PingTimeout       = 5 * time.Second

	RateLimitInterval = 100 * time.Millisecond
	RateLimitBurst    = 10
)

// allowedClientTypes es la lista de tipos de mensaje que un cliente puede enviar.
// Cualquier otro tipo (internos como JOIN, LEAVE, TIMEOUT, SETUP) se rechaza.
var allowedClientTypes = map[string]bool{
	"NAME":          true,
	"READY":         true,
	"CHOOSE_VOWELS": true,
	"SUBMIT":        true,
}

func generateClientID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// ServeWs actualiza la conexión HTTP a WebSocket e inicia el cliente
func ServeWs(room *GameRoom, w http.ResponseWriter, r *http.Request) {
    origin := r.Header.Get("Origin")
    if origin != "" {
        if !isOriginAllowed(origin, r) {
            log.Printf("Bloqueado por Origin mismatch. Recibido: %s", origin)
            http.Error(w, "Origin not allowed", http.StatusForbidden)
            return
        }
    }

    // Aceptar la conexión; InsecureSkipVerify es seguro porque el bloque
    // anterior (isOriginAllowed) ya valida el origen de la petición.
    conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
        InsecureSkipVerify: true,
    })
    if err != nil {
        log.Println("Error upgrade:", err)
        return
    }

	playerID := r.URL.Query().Get("id")
	if playerID == "" {
		playerID = generateClientID()
	}
 
	client := &Client{
		ID:         playerID,
		Conn:       conn,
		SendChan:   make(chan types.ServerMessage, 256),
		Room:       room,
		lastMsgAt:  time.Now(),
	}

	room.safeSendAction(GameAction{
		Type:     "JOIN",
		PlayerID: client.ID,
		Conn:     conn,
		Client:   client,
	})

	go client.readPump()
	go client.writePump()
	go client.pingLoop()
}

// Client engloba la conexión websocket
type Client struct {
	ID         string
	Player     types.Player
	Conn       *websocket.Conn
	SendChan   chan types.ServerMessage
	Room       *GameRoom
	closeOnce  sync.Once
	lastMsgAt  time.Time
	msgTokens  atomic.Int64
}

func (c *Client) closeConn() {
	c.closeOnce.Do(func() {
		c.Conn.Close(websocket.StatusNormalClosure, "")
	})
}

// isRateLimited returns true if the client is sending messages too fast.
// Safe for single-goroutine access (solo llamada desde readPump).
func (c *Client) isRateLimited() bool {
	now := time.Now()
	elapsed := now.Sub(c.lastMsgAt)
	c.lastMsgAt = now

	if elapsed < RateLimitInterval {
		tokens := c.msgTokens.Add(1)
		if tokens > RateLimitBurst {
			return true
		}
	} else {
		c.msgTokens.Store(1)
	}
	return false
}

func (c *Client) readPump() {
	defer func() {
		c.Room.safeSendAction(GameAction{
			Type:     "LEAVE",
			PlayerID: c.ID,
			Client:   c,
		})
		c.closeConn()
	}()

	ctx := context.Background()

	for {
		var msg types.ClientMessage
		err := wsjson.Read(ctx, c.Conn, &msg)
		if err != nil {
			if websocket.CloseStatus(err) != websocket.StatusNormalClosure && websocket.CloseStatus(err) != websocket.StatusGoingAway {
				log.Printf("read error: %v", err)
			}
			break
		}

		if c.isRateLimited() {
			log.Printf("Rate limited client %s", c.ID)
			continue
		}

		if !allowedClientTypes[msg.Type] {
			log.Printf("Rejected message type %q from client %s", msg.Type, c.ID)
			continue
		}

		c.Room.safeSendAction(GameAction{
			Type:     msg.Type,
			PlayerID: c.ID,
			Message:  msg,
		})
	}
}

func (c *Client) writePump() {
	defer c.closeConn()

	for {
		message, ok := <-c.SendChan
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := wsjson.Write(ctx, c.Conn, message)
		cancel()

		if err != nil {
			status := websocket.CloseStatus(err)
			if status == websocket.StatusNormalClosure || status == websocket.StatusGoingAway || strings.Contains(err.Error(), "closed") {
				return
			}
			log.Printf("write error: %v", err)
			return
		}
	}
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), PingTimeout)
		err := c.Conn.Ping(ctx)
		cancel()
		if err != nil {
			c.closeConn()
			return
		}
	}
}

// allowedOriginsCache cachea el parseo de ALLOWED_ORIGINS para no hacerlo por request
var allowedOriginsCache []string
var allowedOriginsOnce sync.Once

func parseAllowedOrigins() []string {
	allowedOriginsOnce.Do(func() {
		env := os.Getenv("ALLOWED_ORIGINS")
		if env == "" {
			allowedOriginsCache = nil
			return
		}
		for _, o := range strings.Split(env, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowedOriginsCache = append(allowedOriginsCache, o)
			}
		}
	})
	return allowedOriginsCache
}

func isOriginAllowed(origin string, r *http.Request) bool {
	// 1. Si hay ALLOWED_ORIGINS, usamos esa lista
	if custom := parseAllowedOrigins(); len(custom) > 0 {
		for _, allowed := range custom {
			if origin == allowed {
				return true
			}
		}
		return false
	}

	// 2. Por defecto: solo mismo origen (same-origin)
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	realHost := r.Header.Get("X-Forwarded-Host")
	if realHost == "" {
		realHost = r.Host
	}
	expectedOrigin := fmt.Sprintf("%s://%s", scheme, realHost)
	if origin == expectedOrigin {
		return true
	}

	// 3. Permitir localhost para desarrollo
	hostOnly := strings.TrimPrefix(strings.TrimPrefix(origin, "http://"), "https://")
	hostOnly = strings.Split(hostOnly, ":")[0]
	if hostOnly == "localhost" || hostOnly == "127.0.0.1" {
		return true
	}

	// 4. Permitir IPs privadas (para testeo en red local)
	ip := net.ParseIP(hostOnly)
	if ip != nil {
		if ip.IsPrivate() {
			return true
		}
	}

	return false
}
