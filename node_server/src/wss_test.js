const { createServer } = require('https')
const { WebSocketServer } = require('ws')
const { readFileSync } = require('fs')

const server = createServer({
    key: readFileSync('../certs/key.pem'),
    cert: readFileSync('../certs/cert.pem')
});
const wss = new WebSocketServer({ server });

wss.on('connection', function connection(ws) {
    ws.on('error', console.error);

    ws.on('message', function message(data) {
        ws.send(data);
    });
});

server.listen(8001, ()=>{
    console.log('WSS Listening to port ' + 8001);
});