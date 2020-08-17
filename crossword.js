let HIGHLIGHTED_CLUE;
let FOCUSED_CELL;
let LAST_CLICKED_CELL;

// e.g. highlightClue("15-across")
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
    if (e.target.tagName != "P" || e.target.classList.contains("highlighted")) return;
    highlightClue(e.target.className)
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
const socket = io();
socket.on('letter', msg => {
    const cell = document.querySelector(`td[data-row='${msg.row}'][data-col='${msg.col}']`);
    cell.querySelector("input").value = msg.key;
    console.log(msg);
})

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
    socket.emit('letter', {key: e.key, row: cell.getAttribute("data-row"), col: cell.getAttribute("data-col")})
    for (let i = 0; i < highlightedCells.length; i++) {
        word.push(highlightedCells[i].querySelector("input").value)
    }
    save(HIGHLIGHTED_CLUE, word)
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
        break
    }
}, false);

function fill(clue, word) {
    const cells = document.querySelectorAll("td." + clue)
    for (let i = 0; i < word.length; i++) {
        cells[i].querySelector("input").value = word[i]
    }
}

function loadAll() {
    const progress = JSON.parse(localStorage.crosswordProgress || "{}");
    const clues = Object.keys(progress);
    for (const clue of clues) {
        fill(clue, progress[clue])
    }
}

function save(clue, attemptedWord) {
    let progress = JSON.parse(localStorage.crosswordProgress || "{}");
    progress[clue] = attemptedWord;
    localStorage.crosswordProgress = JSON.stringify(progress);
}

function load(clue) {
    let progress = JSON.parse(localStorage.crosswordProgress || "{}");
    return progress[clue];
}
window.onload = loadAll
