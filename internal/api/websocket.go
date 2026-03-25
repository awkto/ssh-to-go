package api

import (
	"log"
	"net/http"

	"github.com/awkto/ssh-to-go/internal/relay"
	"nhooyr.io/websocket"
)

func (h *Handlers) WebSocket(w http.ResponseWriter, r *http.Request) {
	hostName := r.PathValue("host")
	sessionName := r.PathValue("session")

	hostCfg, ok := h.Hub.GetHostConfig(hostName)
	if !ok {
		http.Error(w, "host not found", http.StatusNotFound)
		return
	}

	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Allow connections from any origin for local dev
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("websocket accept: %v", err)
		return
	}
	defer ws.CloseNow()

	// Increase read limit for terminal data
	ws.SetReadLimit(64 * 1024)

	log.Printf("terminal attached: host=%s session=%s", hostName, sessionName)

	err = relay.Relay(r.Context(), ws, hostCfg.Address, hostCfg.User, hostCfg.KeyPath, sessionName)
	if err != nil {
		log.Printf("relay error: %v", err)
	}

	ws.Close(websocket.StatusNormalClosure, "session ended")
	log.Printf("terminal detached: host=%s session=%s", hostName, sessionName)
}
