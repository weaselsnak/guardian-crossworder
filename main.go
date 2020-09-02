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
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/websocket"
)

type Point struct {
	X int
	Y int
}

type Clue struct {
	ID        string
	Number    int
	Length    int
	Direction string
	Clue      string
	Position  Point
}

type Crossword struct {
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
		return fmt.Errorf("sorry I am unable to scrape the Guardian: %s", err)
	}
	c, exists := doc.Find(".js-crossword").Attr("data-crossword-data")
	if !exists {
		return fmt.Errorf("sorry I am unable to retrieve crossword data from the Guardian")
	}
	if err := json.Unmarshal([]byte(c), out); err != nil {
		return err
	}
	return nil
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

	for _, entry := range crossword.Entries {
		pos := Point{entry.Position.X, entry.Position.Y}
		for i := 0; i < entry.Length; i++ {
			cell := grid[pos.Y][pos.X]
			if i == 0 {
				cell.Text = strconv.Itoa(entry.Number)
			}
			if cell.Classes == nil {
				cell.Classes = append(cell.Classes, "white")
			}
			cell.Classes = append(cell.Classes, "clue-"+entry.ID)
			if entry.Direction == "across" {
				pos.X++
			} else {
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
	Event string `json:"event"`
	Key   string `json:"key"`
	Row   string `json:"row"`
	Col   string `json:"col"`
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
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
	}
	defer conn.Close()
	addConnection(conn)
	for {
		var msg message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			deleteConnection(conn)
			break
		}
		broadcast <- msg
	}
}

func pingClients() {
	for range time.Tick(25 * time.Second) {
		mu.Lock()
		for conn := range clients {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				deleteConnection(conn)
			}
		}
		mu.Unlock()
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		mu.Lock()
		for conn := range clients {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("error: %v", err)
				conn.Close()
				deleteConnection(conn)
			}
		}
		mu.Unlock()
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
	crosswordType := strings.TrimPrefix(path.Dir(r.URL.Path), "/")
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
	go pingClients()
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
