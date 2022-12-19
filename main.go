package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
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
		"staticpath": func(filename string) string {
			return fmt.Sprintf("/static/%d/%s", versionNo, filename)
		},
	}).ParseFiles("templates/index.gohtml")
	if err != nil {
		log.Fatal(err)
	}
	if err := t.Execute(w, a); err != nil {
		log.Fatal(err)
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

var versionNo = time.Now().Unix()

type sseHandler struct {
	clients *sync.Map
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

type message struct {
	Event string `json:"event,omitempty"`
	Key   string `json:"key,omitempty"`
	Row   string `json:"row,omitempty"`
	Col   string `json:"col,omitempty"`
	Clues string `json:"clues,omitempty"`
}

func main() {
	h := sseHandler{new(sync.Map)}
	prefix := fmt.Sprintf("/static/%d/", versionNo)
	http.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.Dir("static"))))
	http.HandleFunc("/fill", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Printf("unable to read body %s", err)
				return
			}
			id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
			if err != nil {
				log.Printf("unable to parse id %s", err)
				return
			}
			h.Broadcast(string(body), id)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	http.HandleFunc("/", router)
	http.Handle("/stream", h)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Printf("starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

}

var usersConnected int64
var idCounter int64

func (s sseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusBadRequest)
		return
	}
	msgs := make(chan string)
	atomic.AddInt64(&usersConnected, 1)
	atomic.AddInt64(&idCounter, 1)
	s.clients.Store(msgs, atomic.LoadInt64(&idCounter))
	go func() {
		s.Broadcast(fmt.Sprintf(`{"connected": %d}`, atomic.LoadInt64(&usersConnected)), 0)
		msgs <- fmt.Sprintf(`{"id": %d}`, atomic.LoadInt64(&idCounter))
		<-r.Context().Done()
		close(msgs)
		s.clients.Delete(msgs)
		atomic.AddInt64(&usersConnected, -1)
	}()

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	for msg := range msgs {
		fmt.Fprintf(w, "data: %s\n\n", msg)
		f.Flush()
	}
}

func (s sseHandler) Broadcast(event string, sender int64) {
	s.clients.Range(func(client, id any) bool {
		if sender != id {
			client.(chan string) <- event
		}
		return true
	})
}
