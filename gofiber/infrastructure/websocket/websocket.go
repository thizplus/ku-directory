package websocket

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"gofiber-template/pkg/logger"
)

type WebSocketManager struct {
	clients         map[*websocket.Conn]Client
	userConnections map[uuid.UUID]*websocket.Conn // 1 user = 1 connection (prevent duplicates)
	rooms           map[string]map[*websocket.Conn]bool
	register        chan Client
	unregister      chan *websocket.Conn
	broadcast       chan BroadcastMessage
	mutex           sync.RWMutex
}

type Client struct {
	Conn   *websocket.Conn
	UserID uuid.UUID
	RoomID string
}

type Message struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	UserID  string      `json:"userId,omitempty"`
	RoomID  string      `json:"roomId,omitempty"`
}

type BroadcastMessage struct {
	Message Message
	RoomID  string
	UserID  *uuid.UUID
}

var Manager *WebSocketManager

func init() {
	Manager = &WebSocketManager{
		clients:         make(map[*websocket.Conn]Client),
		userConnections: make(map[uuid.UUID]*websocket.Conn),
		rooms:           make(map[string]map[*websocket.Conn]bool),
		register:        make(chan Client),
		unregister:      make(chan *websocket.Conn),
		broadcast:       make(chan BroadcastMessage),
	}
	go Manager.run()
}

func (m *WebSocketManager) run() {
	for {
		select {
		case client := <-m.register:
			m.mutex.Lock()

			// Close old connection if user already has one (prevent duplicates from StrictMode)
			if oldConn, exists := m.userConnections[client.UserID]; exists {
				logger.WebSocketWarn("connection_replaced", "Closing old connection for user (new connection incoming)", map[string]interface{}{"user_id": client.UserID.String()})
				// Clean up old connection from clients map
				if oldClient, ok := m.clients[oldConn]; ok {
					if oldClient.RoomID != "" && m.rooms[oldClient.RoomID] != nil {
						delete(m.rooms[oldClient.RoomID], oldConn)
						if len(m.rooms[oldClient.RoomID]) == 0 {
							delete(m.rooms, oldClient.RoomID)
						}
					}
					delete(m.clients, oldConn)
				}
				oldConn.Close()
			}

			// Register new connection
			m.clients[client.Conn] = client
			m.userConnections[client.UserID] = client.Conn

			if client.RoomID != "" {
				if m.rooms[client.RoomID] == nil {
					m.rooms[client.RoomID] = make(map[*websocket.Conn]bool)
				}
				m.rooms[client.RoomID][client.Conn] = true
			}
			m.mutex.Unlock()

			logger.WebSocket("client_connected", "Client connected (1 connection per user)", map[string]interface{}{"user_id": client.UserID.String(), "room_id": client.RoomID})

		case conn := <-m.unregister:
			m.mutex.Lock()
			if client, ok := m.clients[conn]; ok {
				delete(m.clients, conn)

				// Only remove from userConnections if this is the current connection for this user
				if currentConn, exists := m.userConnections[client.UserID]; exists && currentConn == conn {
					delete(m.userConnections, client.UserID)
				}

				if client.RoomID != "" && m.rooms[client.RoomID] != nil {
					delete(m.rooms[client.RoomID], conn)
					if len(m.rooms[client.RoomID]) == 0 {
						delete(m.rooms, client.RoomID)
					}
				}

				conn.Close()
				logger.WebSocket("client_disconnected", "Client disconnected", map[string]interface{}{"user_id": client.UserID.String(), "room_id": client.RoomID})
			}
			m.mutex.Unlock()

		case message := <-m.broadcast:
			m.mutex.RLock()
			if message.RoomID != "" {
				if clients, ok := m.rooms[message.RoomID]; ok {
					for conn := range clients {
						m.sendMessage(conn, message.Message)
					}
				}
			} else if message.UserID != nil {
				// Use userConnections map for direct lookup (1 message per user)
				if conn, exists := m.userConnections[*message.UserID]; exists {
					m.sendMessage(conn, message.Message)
				}
			} else {
				for conn := range m.clients {
					m.sendMessage(conn, message.Message)
				}
			}
			m.mutex.RUnlock()
		}
	}
}

func (m *WebSocketManager) sendMessage(conn *websocket.Conn, message Message) {
	if err := conn.WriteJSON(message); err != nil {
		logger.WebSocketError("send_message", "Error sending message", err, map[string]interface{}{"message_type": message.Type})
		m.unregister <- conn
	}
}

func (m *WebSocketManager) RegisterClient(conn *websocket.Conn, userID uuid.UUID, roomID string) {
	client := Client{
		Conn:   conn,
		UserID: userID,
		RoomID: roomID,
	}
	m.register <- client
}

func (m *WebSocketManager) UnregisterClient(conn *websocket.Conn) {
	m.unregister <- conn
}

func (m *WebSocketManager) BroadcastToRoom(roomID string, messageType string, data interface{}) {
	message := Message{
		Type: messageType,
		Data: data,
	}

	broadcast := BroadcastMessage{
		Message: message,
		RoomID:  roomID,
	}

	m.broadcast <- broadcast
}

func (m *WebSocketManager) BroadcastToUser(userID uuid.UUID, messageType string, data interface{}) {
	logger.WebSocketDebug("broadcast_to_user", "Broadcasting to user", map[string]interface{}{"user_id": userID.String(), "message_type": messageType})

	message := Message{
		Type: messageType,
		Data: data,
	}

	broadcast := BroadcastMessage{
		Message: message,
		UserID:  &userID,
	}

	m.broadcast <- broadcast
}

func (m *WebSocketManager) BroadcastToAll(messageType string, data interface{}) {
	message := Message{
		Type: messageType,
		Data: data,
	}

	broadcast := BroadcastMessage{
		Message: message,
	}

	m.broadcast <- broadcast
}

func (m *WebSocketManager) GetRoomClients(roomID string) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if clients, ok := m.rooms[roomID]; ok {
		return len(clients)
	}
	return 0
}

func (m *WebSocketManager) GetTotalClients() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.clients)
}

func HandleWebSocketMessage(conn *websocket.Conn, messageType int, data []byte) {
	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		logger.WebSocketError("unmarshal_message", "Error unmarshaling message", err, nil)
		return
	}

	switch message.Type {
	case "ping":
		response := Message{
			Type: "pong",
			Data: "pong",
		}
		conn.WriteJSON(response)

	case "join_room":
		if roomData, ok := message.Data.(map[string]interface{}); ok {
			if roomID, ok := roomData["roomId"].(string); ok {
				Manager.mutex.Lock()
				if client, exists := Manager.clients[conn]; exists {
					if client.RoomID != "" && Manager.rooms[client.RoomID] != nil {
						delete(Manager.rooms[client.RoomID], conn)
						if len(Manager.rooms[client.RoomID]) == 0 {
							delete(Manager.rooms, client.RoomID)
						}
					}

					client.RoomID = roomID
					Manager.clients[conn] = client

					if Manager.rooms[roomID] == nil {
						Manager.rooms[roomID] = make(map[*websocket.Conn]bool)
					}
					Manager.rooms[roomID][conn] = true
				}
				Manager.mutex.Unlock()

				response := Message{
					Type: "room_joined",
					Data: map[string]interface{}{
						"roomId": roomID,
						"message": fmt.Sprintf("Joined room %s", roomID),
					},
				}
				conn.WriteJSON(response)
			}
		}

	case "leave_room":
		Manager.mutex.Lock()
		if client, exists := Manager.clients[conn]; exists {
			if client.RoomID != "" && Manager.rooms[client.RoomID] != nil {
				delete(Manager.rooms[client.RoomID], conn)
				if len(Manager.rooms[client.RoomID]) == 0 {
					delete(Manager.rooms, client.RoomID)
				}

				client.RoomID = ""
				Manager.clients[conn] = client
			}
		}
		Manager.mutex.Unlock()

		response := Message{
			Type: "room_left",
			Data: "Left room successfully",
		}
		conn.WriteJSON(response)

	default:
		logger.WebSocketWarn("unknown_message_type", "Unknown message type received", map[string]interface{}{"message_type": message.Type})
	}
}