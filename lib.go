package hxsocketsfiber

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type ListenerFunc func(*Client, []byte)
type ClientConnectFunc func(*Client)
type ClientDisconnectFunc func(*Client)
type Server struct {
	app                *fiber.App
	clients            map[string]*Client
	listeners          map[string]ListenerFunc
	OnClientConnect    ClientConnectFunc
	OnClientDisconnect ClientDisconnectFunc
	mtex               sync.Mutex
}

func (s *Server) GetAllClients() []*Client {
	ret := []*Client{}
	for _, client := range s.clients {
		ret = append(ret, client)
	}
	return ret
}

func (s *Server) GetClient(id string) *Client {
	c, ok := s.clients[id]
	if !ok {
		return nil
	}
	return c
}

func (s *Server) GetClientFilter(filter func(*Client) bool) []*Client {
	ret := []*Client{}
	for _, c := range s.clients {
		if filter(c) {
			ret = append(ret, c)
		}
	}

	return ret
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (s *Server) Listen(endpoint string, handler ListenerFunc) error {
	s.mtex.Lock()
	defer s.mtex.Unlock()

	_, exists := s.listeners[endpoint]
	if exists {
		return fmt.Errorf("endpoint %s already registered", endpoint)
	}
	s.listeners[endpoint] = handler

	return nil
}

func (s *Server) Mount(endpoint string) {
	s.app.Use(endpoint, func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	s.app.Get(endpoint, websocket.New(func(c *websocket.Conn) {
		allowed, ok := c.Locals("allowed").(bool)

		if !ok {
			log.Println("c.locals not found")
			return
		}

		if !allowed {
			log.Println("not allowed")
			return
		}

		newClient := Client{
			conn: c,
			id:   GenB64(10),
		}

		s.clients[newClient.id] = &newClient

		log.Printf("client connected %+v", newClient)

		for {
			_, msg, err := c.ReadMessage()
			hd := HXWSHeaders{}

			err = json.Unmarshal(msg, &hd)
			if err != nil {
				log.Printf("Failed to unmarshal json %+v", err)
				return
			}
			log.Printf(string(msg))
			if len(hd.Headers.HXTrigger) == 0 {
				log.Printf("Trigger was blank %+v", hd)
				return
			}
			listener, ok := s.listeners[hd.Headers.HXTrigger]
			if !ok {
				log.Default().Println("no listener found for endpoint ", hd.Headers.HXTrigger)
				return
			}

			listener(&newClient, msg)
			if err != nil {
				log.Println("read:", err)
				break

			}

		}
	}))

}

type Client struct {
	conn *websocket.Conn
	id   string
}

func (c *Client) WriteMessage(code int, msg []byte) error {
	return c.conn.WriteMessage(code, msg)
}

func NewServer(app *fiber.App) Server {
	return Server{
		app:                app,
		clients:            map[string]*Client{},
		listeners:          map[string]ListenerFunc{},
		OnClientConnect:    func(*Client) {},
		OnClientDisconnect: func(*Client) {},
		mtex:               sync.Mutex{},
	}
}
func GenB64(length int) string {
	dembytes := make([]byte, length)
	_, err := rand.Read(dembytes)
	if err != nil {
		return ""
	}
	encoded := base64.URLEncoding.EncodeToString(dembytes)
	return encoded
}

type HXWSHeaders struct {
	Headers struct {
		HXRequest     string  `json:"HX-Request"`
		HXTrigger     string  `json:"HX-Trigger"`
		HXTriggerName *string `json:"HX-Trigger-Name"`
		HXTarget      string  `json:"HX-Target"`
		HXCurrentURL  string  `json:"HX-Current-URL"`
	} `json:"HEADERS"`
}
