let HIGHLIGHTED_CLUE;
let FOCUSED_CELL;
let LAST_CLICKED_CELL;
let ID;
const CROSSWORD_ID = document.getElementById("crossword-id").value; // crosswords/quick/1234


async function send(payload) {
	const response = await fetch(
		`/fill?id=${ID}`,
		{
			method: "POST",
			body: payload
		}
	);
	if (!response.ok) {
		throw new Error(`HTTP error! status: ${response.status}`);
	}
}

// e.g. highlightClue("clue-15-across")
function highlightClue(clue) {
    if (HIGHLIGHTED_CLUE) {
        const cells = document.querySelectorAll("." + HIGHLIGHTED_CLUE)
        cells.forEach(c => c.classList.remove("highlighted"))
    }
    const cells = document.querySelectorAll("." + clue)
    cells.forEach(c => c.classList.add("highlighted"))
    HIGHLIGHTED_CLUE = clue;
}

document.querySelector('aside').addEventListener('click', e => {
    const div = e.target.parentNode;
    if (e.target.tagName != "LI" || div.classList.contains("highlighted")) return;
    highlightClue(div.className)
    const firstInput = document.querySelector("td.highlighted input")
    firstInput.select()
}, false);


function moveFocus(cells, cell, offset) {
    for (let i = 0; i < cells.length; i++) {
        if (cells[i] == cell && cells[i + offset]) {
            setTimeout(() => {
                cells[i + offset].querySelector("input").select();
            }, 0);
            break;
        }
    }
}

function click(cells, cell, offset) {
    for (let i = 0; i < cells.length; i++) {
        if (cells[i] == cell && cells[i + offset]) {
            setTimeout(() => {
                cells[i + offset].querySelector("input").click();
                cells[i + offset].querySelector("input").select();
            }, 0);
            break;
        }
    }
}


var es = new EventSource('/stream');
es.onmessage = function (e) {
	const msg = JSON.parse(e.data);
	if (msg.connected) {
		document.getElementById("connected").innerHTML = msg.connected;
		return
	}
	if (msg.id) {
		ID = msg.id;
		return
	}
	const cell = document.querySelector(`td[data-row='${msg.row}'][data-col='${msg.col}']`);
	const cells = document.querySelectorAll("td.white")
	if (msg.event == 'letter') {
		cell.querySelector("input").value = msg.key;
		save(cells)
		return;
	}
	if (msg.event == 'backspace') {
		save(cells)
		cell.querySelector("input").value = "";
		return;
	}
};


document.querySelector('table').addEventListener('keydown', e => {
    const cell = e.target.closest('td.white')
    if (!cell) return;
    const LEFT_ARROW = 37;
    const UP_ARROW = 38;
    const RIGHT_ARROW = 39;
    const DOWN_ARROW = 40;
    let direction;
    let offset;
    if (e.keyCode == LEFT_ARROW) {
        direction = "-across";
        offset = -1;
    } else if (e.keyCode == RIGHT_ARROW) {
        direction = "-across";
        offset = 1;
    } else if (e.keyCode == UP_ARROW) {
        direction = "-down";
        offset = -1; 
    } else if (e.keyCode == DOWN_ARROW) {
        direction = "-down";
        offset = 1; 
    }
    let clue = Array.from(cell.classList).filter(c => c.match(direction))[0]
    if (!clue) return;
    const cells = document.querySelectorAll("td." + clue)
    click(cells, cell, offset)
}, false);

document.querySelector('table').addEventListener('keypress', e => {
    if (e.key < 'a' || e.key > 'z') return e.preventDefault();
    const cell = e.target.closest('td.white')
    if (!cell) return;
    const highlightedCells = document.querySelectorAll("td.highlighted")
    let word = []
    cell.querySelector("input").value = e.key
    const cells = document.querySelectorAll("td.white")
    save(cells)
    send(JSON.stringify({event: 'letter', key: e.key, row: cell.getAttribute("data-row"), col: cell.getAttribute("data-col"), clues: HIGHLIGHTED_CLUE}))
    for (let i = 0; i < highlightedCells.length; i++) {
        word.push(highlightedCells[i].querySelector("input").value)
    }
    moveFocus(highlightedCells, cell, 1)
    e.preventDefault();
}, false);

document.querySelector('table').addEventListener('keyup', e => {
    const BACKSPACE_KEY = 8
    if (e.keyCode != BACKSPACE_KEY) {
        return
    }
    const cell = e.target.closest('td.white')
    if (!cell) return;
    const highlightedCells = document.querySelectorAll("td.highlighted")
    send(JSON.stringify({event: 'backspace', row: cell.getAttribute("data-row"), col: cell.getAttribute("data-col")})) 
    moveFocus(highlightedCells, cell, -1)
}, false);

document.querySelector('table').addEventListener('focusin', e => {
    const td = e.target.closest('td.white');
    if (!td) return;
    if (FOCUSED_CELL) {
        FOCUSED_CELL.classList.remove("focused")
    }
    td.classList.add("focused")
    FOCUSED_CELL = td;
}, false);

document.querySelector('table').addEventListener('click', e => {
    const td = e.target.closest('td.white');
    if (!td) return;
    const alreadyHighlighted = td.classList.contains('highlighted');
    if (alreadyHighlighted && LAST_CLICKED_CELL != td) {
        LAST_CLICKED_CELL = td;
        return
    }
    LAST_CLICKED_CELL = td;
    for (let i = 1; i < td.classList.length; i++) {
        const clue = td.classList[i]
        if (!clue.match("clue-")) {
            continue
        }
        if (HIGHLIGHTED_CLUE == clue) {
            continue
        }
        highlightClue(clue)
        send(JSON.stringify({event: "click", clues: clue}));
        break
    }
}, false);

function fill(cellData) {
    const cell = document.querySelector(`td[data-row='${cellData.row}'][data-col='${cellData.col}']`);
    cell.querySelector("input").value = cellData.letter;
}

function loadAll() {
    const progress = JSON.parse(localStorage.crosswordProgress || "{}");
    if (!progress[CROSSWORD_ID]) {
        return
    }
    let crosswordData = progress[CROSSWORD_ID];
    for (const pos of Object.keys(crosswordData)) {
        fill(crosswordData[pos])
    }
}

function save(cells) {
    let progress = JSON.parse(localStorage.crosswordProgress || "{}");
    progress[CROSSWORD_ID] ||= {};
    for (let i = 0; i < cells.length; i++) {
        let row = cells[i].getAttribute("data-row");
        let col = cells[i].getAttribute("data-col");
        let letter = cells[i].querySelector("input").value;
        progress[CROSSWORD_ID][row+col] = {letter: letter, row: row, col: col}
    }
    localStorage.crosswordProgress = JSON.stringify(progress);
}

window.onload = loadAll
