package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
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

func getCrossword(num int, out interface{}) error {
	url := fmt.Sprintf("https://www.theguardian.com/crosswords/quick/%d", num)
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

func main() {
	var crosswordNumber int
	var crosswordType string
	flag.IntVar(&crosswordNumber, "n", 1234, "please pass in a crossword number")
	flag.StringVar(&crosswordType, "type", "quick", "quick, cryptic, quiptic etc")
	flag.Parse()

	var crossword Crossword
	if err := getCrossword(crosswordNumber, &crossword); err != nil {
		log.Fatal(err)
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

	fmt.Println(`<link rel="stylesheet" href="style.css">`)
	fmt.Println(`<header>`)
	fmt.Println(`<h2>` + crossword.Name + `</h2>`)
	fmt.Println(`</header>`)
	fmt.Println(`<main>`)
	fmt.Println("<article>")
	fmt.Println(`<table cellspacing="0">`)
	for i, row := range grid {
		fmt.Println("<tr>")
		for j, cell := range row {
			if len(cell.Classes) > 0 {
				fmt.Printf(`<td data-row=%d data-col=%d class=%q><span class="clue-number">%s</span><input type="text" maxlength="1"></td>`, i, j, strings.Join(cell.Classes, " "), cell.Text)
			} else {
				fmt.Printf(`<td data-row=%d data-col=%d class=%q><span class="clue-number">%s</span></td>`, i, j, strings.Join(cell.Classes, " "), cell.Text)

			}
		}
		fmt.Println("\n</tr>")
	}
	fmt.Println("</table>")
	fmt.Println("</article>")

	fmt.Println(`<aside>`)
	fmt.Println(`<section><h2>Across</h2>`)
	for _, entry := range crossword.Entries {
		if entry.Direction != "across" {
			continue
		}
		fmt.Printf(`<p class="clue-%s">%d. %s</p>`, entry.ID, entry.Number, entry.Clue)
	}
	fmt.Println(`</section>`)
	fmt.Println(`<section><h2>Down</h2>`)
	for _, entry := range crossword.Entries {
		if entry.Direction != "down" {
			continue
		}
		fmt.Printf(`<p class="clue-%s">%d. %s</p>`, entry.ID, entry.Number, entry.Clue)
	}
	fmt.Println(`</section>`)
	fmt.Println(`</aside>`)
	fmt.Println(`</main>`)

	fmt.Println(`<script src="/socket.io/socket.io.js"></script>`)
	fmt.Println(`<script src="crossword.js"></script>`)
}
