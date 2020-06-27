
# Setup

Regenerate index.html for a different crossword by downloading the crossword data into `testdata/guardian.json`, and then:

	go run cmd/parse/parse.go < testdata/guardian.json > index.html

Have a look in your browser. Run `highlightClue("4-down")` or something like that.

# Syllabus

- Add a click event that highlights the appropriate clue
