package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"text/template"

	sockets "github.com/deastl/hxsocketsfiber"
	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	server := sockets.NewServer(app)
	server.Mount("/ws")

	server.Listen("some_message", func(client *sockets.Client, msg []byte) {
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
		err = client.WriteMessage(1, buf.Bytes())
		if err != nil {
			fmt.Println(err.Error())
		}
	})

	app.Static("/", "./static/", fiber.Static{})
	log.Fatal(app.Listen(":3000"))
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
