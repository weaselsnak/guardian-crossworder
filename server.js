var express = require('express');
var app = express();
var http = require('http').createServer(app);
var io = require('socket.io')(http);
const port = process.env.PORT || 5000;

app.use('/', express.static(__dirname));

io.on('connection', (socket) => {
    console.log('a user connected');
    socket.on('disconnect', () => {
        console.log('user disconnected');
    });
    socket.on('letter', (msg) => {
        socket.broadcast.emit('letter', msg);
    });
    socket.on('backspace', (msg) => {
        socket.broadcast.emit('backspace', msg);
    });
});


http.listen(port, () => {
  console.log(`listening on *:${port}`);
});
