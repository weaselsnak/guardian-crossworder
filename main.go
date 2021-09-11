package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/websocket"
)

type Point struct {
	X int
	Y int
}

type Clue struct {
	ID                 string
	Number             int
	Length             int
	Direction          string
	Clue               template.HTML
	Position           Point
	SeparatorLocations map[string][]int
	Group              []string // if the clue spans across mulitple down/accross on the grid
}

type Crossword struct {
	Id         string // is of the form crosswords/quick/15655
	Name       string
	Dimensions struct {
		Rows int
		Cols int
	}
	Entries []Clue
}

type Cell struct {
	Text    string
	Classes []string
}

type All struct {
	Crossword Crossword
	Grid      [][]*Cell
}

func getCrossword(crosswordType string, num int, out interface{}) error {
	url := fmt.Sprintf("https://www.theguardian.com/crosswords/%s/%d", crosswordType, num)
	log.Printf("getting crossword %s", url)
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return fmt.Errorf("sorry I am unable to scrape the Guardian crossword: %s", err)
	}
	c, exists := doc.Find(".js-crossword").Attr("data-crossword-data")
	if !exists {
		return fmt.Errorf("sorry I am unable to retrieve the Guardian crossword")
	}
	if err := json.Unmarshal([]byte(c), out); err != nil {
		return err
	}
	return nil
}
func separatorPositions(s map[string][]int) []int {
	for separator, numbers := range s {
		switch separator {
		case ",":
			return numbers
		case "-":
			return numbers
		}
	}
	return nil
}

func clueGroups(groups []string) []string {
	var classes []string
	for _, g := range groups {
		classes = append(classes, "clue-"+g)
	}
	return classes
}

var crosswordTypes = []string{"quick", "cryptic", "prize", "weekend", "quiptic", "genius", "speedy", "everyman"}

func generateCrossword(w http.ResponseWriter, r *http.Request, crosswordType string, crosswordNumber int) {
	var crossword Crossword
	if err := getCrossword(crosswordType, crosswordNumber, &crossword); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	grid := make([][]*Cell, crossword.Dimensions.Rows)

	for i := range grid {
		grid[i] = make([]*Cell, crossword.Dimensions.Cols)
		for j := range grid[i] {
			grid[i][j] = &Cell{}
		}
	}

	for j, entry := range crossword.Entries {
		pos := Point{entry.Position.X, entry.Position.Y}
		crossword.Entries[j].Group = clueGroups(entry.Group)
		for i := 0; i < entry.Length; i++ {
			cell := grid[pos.Y][pos.X]
			if i == 0 {
				cell.Text = strconv.Itoa(entry.Number)
			}
			if cell.Classes == nil {
				cell.Classes = append(cell.Classes, "white")
			}
			cell.Classes = append(cell.Classes, clueGroups(entry.Group)...)
			if entry.Direction == "across" {
				if len(entry.SeparatorLocations) != 0 {
					positions := separatorPositions(entry.SeparatorLocations)
					for _, p := range positions {
						if p == i+1 {
							cell.Classes = append(cell.Classes, "sep-across")
						}
					}
				}
				pos.X++
			} else {
				if len(entry.SeparatorLocations) != 0 {
					positions := separatorPositions(entry.SeparatorLocations)
					for _, p := range positions {
						if p == i+1 {
							cell.Classes = append(cell.Classes, "sep-down")
						}
					}
				}
				pos.Y++
			}
		}
	}
	var a All
	a.Grid = grid
	a.Crossword = crossword
	t, err := template.New("index.gohtml").Funcs(template.FuncMap{
		"join": func(s []string) string {
			return strings.Join(s, " ")
		},
	}).ParseFiles("templates/index.gohtml")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, a); err != nil {
		log.Fatal(err)
	}
}

type message struct {
	Event     string          `json:"event"`
	Key       string          `json:"key"`
	Row       string          `json:"row"`
	Col       string          `json:"col"`
	Clues     string          `json:"clues"`
	Connected int64           `json:"connected"`
	Sender    *websocket.Conn `json:"sender"`
}

var broadcast = make(chan message)
var mu = sync.Mutex{}
var clients = make(map[*websocket.Conn]bool)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func deleteConnection(conn *websocket.Conn) {
	mu.Lock()
	delete(clients, conn)
	mu.Unlock()

}
func addConnection(conn *websocket.Conn) {
	mu.Lock()
	clients[conn] = true
	mu.Unlock()
}

func closeGracefully(conn *websocket.Conn, err error) {
	log.Println(err)
	atomic.AddInt64(&usersConnected, -1)
	conn.Close()
	deleteConnection(conn)
	broadcast <- message{Connected: atomic.LoadInt64(&usersConnected)}
}

var usersConnected int64

func getConnections() (conns []*websocket.Conn) {
	mu.Lock()
	for conn := range clients {
		conns = append(conns, conn)
	}
	mu.Unlock()
	return conns
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
	}
	defer conn.Close()
	addConnection(conn)
	atomic.AddInt64(&usersConnected, 1)
	broadcast <- message{Connected: atomic.LoadInt64(&usersConnected)}
	for {
		var msg message
		if err := conn.ReadJSON(&msg); err != nil {
			closeGracefully(conn, err)
			break
		}
		if msg == (message{}) {
			continue
		}
		msg.Sender = conn
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		select {
		case msg := <-broadcast:
			conns := getConnections()
			for _, conn := range conns {
				// we want to skip writing messages to the sender
				if msg.Sender == conn {
					continue
				}
				if err := conn.WriteJSON(msg); err != nil {
					closeGracefully(conn, err)
				}
			}
		case <-time.After(30 * time.Second):
			conns := getConnections()
			for _, conn := range conns {
				if err := conn.WriteJSON(message{}); err != nil {
					closeGracefully(conn, err)
				}
			}
		}
	}
}

func isValid(crosswordType string) bool {
	for _, c := range crosswordTypes {
		if strings.EqualFold(c, crosswordType) {
			return true
		}
	}
	return false
}

func router(w http.ResponseWriter, r *http.Request) {
	crosswordType := strings.ToLower(strings.TrimPrefix(path.Dir(r.URL.Path), "/"))
	if !isValid(crosswordType) {
		http.NotFound(w, r)
		return
	}
	crosswordNumber, err := strconv.Atoi(path.Base(r.URL.Path))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	generateCrossword(w, r, crosswordType, crosswordNumber)
}

func main() {
	http.HandleFunc("/ws", wsHandler)
	http.Handle("/static/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/", router)
	go handleMessages()
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
