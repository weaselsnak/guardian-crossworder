var express = require('express');
var app = express();
var http = require('http').createServer(app);
var io = require('socket.io')(http);

app.use('/', express.static(__dirname));

io.on('connection', (socket) => {
    console.log('a user connected');
    socket.on('disconnect', () => {
        console.log('user disconnected');
    });
    socket.on('letter', (msg) => {
        socket.broadcast.emit('letter', msg);
        console.log('letter: ' + msg);
    });
});


http.listen(3000, () => {
  console.log('listening on *:3000');
});
