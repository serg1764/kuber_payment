package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSDemoHandler struct {
	upgrader websocket.Upgrader

	mu          sync.RWMutex
	clients     map[*websocket.Conn]struct{}
	currentPair string
}

type wsInboundMessage struct {
	Type      string `json:"type"`
	Pair      string `json:"pair,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

type wsQuoteMessage struct {
	Type  string  `json:"type"`
	Pair  string  `json:"pair"`
	Price float64 `json:"price"`
	TS    string  `json:"ts"`
}

type wsAckMessage struct {
	Type      string `json:"type"`
	Status    string `json:"status"`
	Pair      string `json:"pair,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

type changePairRequest struct {
	Pair string `json:"pair"`
}

// NewWSDemoHandler создает демо-обработчик WebSocket в памяти с парой по умолчанию.
func NewWSDemoHandler() *WSDemoHandler {
	return &WSDemoHandler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(_ *http.Request) bool {
				return true
			},
		},
		clients:     make(map[*websocket.Conn]struct{}),
		currentPair: "EURUSD",
	}
}

// WS переключает HTTP-соединение в WebSocket и запускает циклы чтения/записи.
func (h *WSDemoHandler) WS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	// Если upgrade не удался, продолжать обработку WebSocket нельзя.
	if err != nil {
		log.Printf("ws upgrade failed: %v", err)
		return
	}

	h.addClient(conn)
	log.Printf("ws connected remote=%s clients=%d", r.RemoteAddr, h.clientCount())

	go h.writeLoop(conn)
	h.readLoop(conn)
}

// ChangePair отправляет команду change_pair подключенным WebSocket-клиентам через HTTP.
func (h *WSDemoHandler) ChangePair(w http.ResponseWriter, r *http.Request) {
	var req changePairRequest

	// Если тело запроса невалидный JSON, отклоняем команду.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	pair := strings.ToUpper(strings.TrimSpace(req.Pair))

	// Если после нормализации пара пустая, отклоняем команду.
	if pair == "" {
		writeError(w, http.StatusBadRequest, "pair is required")
		return
	}

	requestID := "http-" + strconvTime()
	cmd := wsInboundMessage{
		Type:      "change_pair",
		Pair:      pair,
		RequestID: requestID,
	}

	// Сначала обновляем текущую пару на сервере, чтобы поток quote переключился сразу.
	h.mu.Lock()
	h.currentPair = pair
	h.mu.Unlock()

	payload, err := json.Marshal(cmd)

	// Если сериализация не удалась, не можем отправить команду клиентам.
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build message")
		return
	}

	sent := h.broadcastText(payload)

	// Если нет подключенных клиентов, пару уже сменили, но уведомление никому не доставили.
	if sent == 0 {
		log.Printf("http pair changed without active ws clients pair=%s requestId=%s", pair, requestID)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "updated_without_clients",
			"pair":      pair,
			"requestId": requestID,
			"delivered": 0,
		})
		return
	}

	log.Printf("http requested pair change pair=%s requestId=%s delivered=%d", pair, requestID, sent)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "sent",
		"pair":      pair,
		"requestId": requestID,
		"delivered": sent,
	})
}

// Status возвращает текущее состояние демо WebSocket для быстрой проверки.
func (h *WSDemoHandler) Status(w http.ResponseWriter, _ *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"connected":    len(h.clients) > 0,
		"clients":      len(h.clients),
		"current_pair": h.currentPair,
	})
}

// readLoop непрерывно принимает сообщения клиента и применяет поддерживаемые команды.
func (h *WSDemoHandler) readLoop(conn *websocket.Conn) {
	defer h.removeClient(conn)

	// Читаем до тех пор, пока сокет не закроется или чтение не завершится ошибкой.
	for {
		_, data, err := conn.ReadMessage()

		// Если чтение упало, считаем соединение закрытым/поврежденным.
		if err != nil {
			log.Printf("ws read error: %v", err)
			return
		}

		log.Printf("ws received raw=%s", string(data))

		var msg wsInboundMessage

		// Если payload невалидный JSON, пропускаем этот кадр и читаем дальше.
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("ws invalid json: %v", err)
			continue
		}

		// Если команда change_pair, валидируем данные и обновляем общую текущую пару.
		if msg.Type == "change_pair" {
			pair := strings.ToUpper(strings.TrimSpace(msg.Pair))
			// Если пара невалидна, возвращаем error ack для этого request id.
			if pair == "" {
				h.writeJSON(conn, wsAckMessage{
					Type:      "ack",
					Status:    "error",
					RequestID: msg.RequestID,
				})
				continue
			}

			h.mu.Lock()
			h.currentPair = pair
			h.mu.Unlock()

			log.Printf("ws pair changed pair=%s requestId=%s", pair, msg.RequestID)
			h.writeJSON(conn, wsAckMessage{
				Type:      "ack",
				Status:    "ok",
				Pair:      pair,
				RequestID: msg.RequestID,
			})
		}
	}
}

// writeLoop отправляет этому WebSocket-клиенту синтетическую котировку раз в секунду.
func (h *WSDemoHandler) writeLoop(conn *websocket.Conn) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// На каждом тике публикуем котировку текущей пары, пока запись не упадет.
	for range ticker.C {
		h.mu.RLock()
		pair := h.currentPair
		h.mu.RUnlock()

		quote := wsQuoteMessage{
			Type:  "quote",
			Pair:  pair,
			Price: fakePrice(pair),
			TS:    time.Now().UTC().Format(time.RFC3339),
		}

		// Если запись не удалась, останавливаем цикл записи для этого соединения.
		if err := h.writeJSON(conn, quote); err != nil {
			return
		}
	}
}

// writeJSON сериализует payload и отправляет его как текстовый WebSocket-кадр.
func (h *WSDemoHandler) writeJSON(conn *websocket.Conn, payload any) error {
	data, err := json.Marshal(payload)

	// Если сериализация не удалась, вызывающий код получает ошибку и решает retry/abort.
	if err != nil {
		return err
	}

	// Если запись не удалась, удаляем клиента из реестра и возвращаем ошибку.
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("ws write error: %v", err)
		h.removeClient(conn)
		return err
	}
	return nil
}

// broadcastText отправляет один и тот же текстовый кадр всем подключенным клиентам.
func (h *WSDemoHandler) broadcastText(payload []byte) int {
	h.mu.RLock()
	clients := make([]*websocket.Conn, 0, len(h.clients))

	// Снимаем snapshot клиентов под read-lock, чтобы не держать lock во время сетевой записи.
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	sent := 0

	// Пытаемся доставить сообщение каждому клиенту и удаляем нерабочие соединения.
	for _, c := range clients {

		// Если запись клиенту не удалась, удаляем его и продолжаем со следующими.
		if err := c.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Printf("ws broadcast write error: %v", err)
			h.removeClient(c)
			continue
		}
		sent++
	}
	return sent
}

// addClient регистрирует новое подключенное сокет-соединение.
func (h *WSDemoHandler) addClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[conn] = struct{}{}
}

// removeClient удаляет сокет из реестра и закрывает соединение.
func (h *WSDemoHandler) removeClient(conn *websocket.Conn) {
	h.mu.Lock()

	// Если соединение есть в реестре, удаляем его один раз.
	if _, ok := h.clients[conn]; ok {
		delete(h.clients, conn)
	}
	remaining := len(h.clients)
	h.mu.Unlock()

	_ = conn.Close()
	log.Printf("ws disconnected clients=%d", remaining)
}

// clientCount возвращает количество активных websocket-клиентов.
func (h *WSDemoHandler) clientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// fakePrice генерирует демо-цену с базой, зависящей от валютной пары.
func fakePrice(pair string) float64 {
	base := 1.1

	// Выбираем разные базовые значения для известных валютных пар.
	switch pair {
	case "GBPUSD":
		base = 1.27
	case "USDJPY":
		base = 156.3
	case "EURUSD":
		base = 1.08
	}
	noise := float64(time.Now().UnixNano()%1000) / 100000.0
	return base + noise
}

// strconvTime возвращает компактную UTC-метку времени для request id.
func strconvTime() string {
	return time.Now().UTC().Format("20060102T150405.000000000")
}
