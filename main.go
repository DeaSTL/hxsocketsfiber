package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"text/template"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	server := NewServer(app)
	server.Mount("/ws")

	server.Listen("some_message", func(client *Client, msg []byte) {
		message := recvData{}
		println(string(msg))
		err := json.Unmarshal(msg, &message)
		if err != nil {
			fmt.Println(err.Error())
		}
		buf := bytes.NewBuffer([]byte{})
		t, err := template.New("").Parse(buttonTemplate)
		if err != nil {
			fmt.Println("error unmarshaling")
		}
		fmt.Printf("message.on is %v", message.On)
		t.Execute(buf, !message.On)
		fmt.Println()
		os.Stdout.Write(buf.Bytes())
		err = client.conn.WriteMessage(1, buf.Bytes())
		if err != nil {
			fmt.Println(err.Error())
		}
	})

	app.Static("/", "./static/", fiber.Static{})
	log.Fatal(app.Listen(":3000"))
}

type ListenerFunc func(*Client, []byte)
type Server struct {
	app       *fiber.App
	clients   map[string]*Client
	listeners map[string]ListenerFunc
	mtex      sync.Mutex
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

func NewServer(app *fiber.App) Server {
	return Server{
		app:       app,
		clients:   map[string]*Client{},
		listeners: map[string]ListenerFunc{},
		mtex:      sync.Mutex{},
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

type recvData struct {
	On bool `json:"state"`
}

var buttonTemplate string = `<button
		id="some_message"
		hx-vals='{"state": {{if .}}true{{else}}false{{end}}}'
		hx-trigger="click"
		ws-send
		>
		{{if .}}on{{else}}off{{end}}
	</button>`
