var net = require('net');
var uws = require('uWebSockets.js')

const { WebSocketServer } = require('ws')
const { readFileSync } = require('fs')
const { createServer } = require('https')

var HOST = 'localhost';
var TCP_PORT = 8000;
var WS_PORT = 8001;

function startTCP() {
    // Create a server instance, and chain the listen function to it
    // The function passed to net.createServer() becomes the event handler for the 'connection' event
    // The sock object the callback function receives UNIQUE for each connection
    net.createServer(function(sock) {
        // We have a connection - a socket object is assigned to the connection automatically
        console.log('CONNECTED: ' + sock.remoteAddress +':'+ sock.remotePort);
        // Add a 'data' event handler to this instance of socket
        sock.on('data', function(data) {
            console.log('DATA ' + sock.remoteAddress + ': ' + data + ' ' + typeof(data));
            // Write the data back to the socket, the client will receive it as data from the server
            //sock.write('You said "' + data + '"');

            let arr = new Uint8Array(2);
            arr[0] = 111;
            arr[1] = 222;

            let param = new Uint8Array(data)
            console.log('DATA ' + param[0] + ', ' + param[1])
            arr = Buffer.concat([arr, param], arr.length + param.length);
            console.log('arr ' + arr[0] + ', ' + arr[1] + ', ' + arr[2] + ', ' + arr[3])
            console.log('length ' + arr.length)
        });
        // Add a 'close' event handler to this instance of socket
        sock.on('close', function(data) {
            console.log('CLOSED: ' + sock.remoteAddress +' '+ sock.remotePort);
        });

    }).listen(TCP_PORT, HOST);

    console.log('Server listening on ' + HOST +':'+ TCP_PORT);
}

function startWS() {

    console.log('start WS');

    const perMessageDeflateOption = {
        zlibDeflateOptions: {
          // See zlib defaults.
          chunkSize: 1024,
          memLevel: 7,
          level: 3
        },
        zlibInflateOptions: {
          chunkSize: 10 * 1024
        },
        // Other options settable:
        clientNoContextTakeover: true, // Defaults to negotiated value.
        serverNoContextTakeover: true, // Defaults to negotiated value.
        serverMaxWindowBits: 10, // Defaults to negotiated value.
        // Below options specified as default values.
        concurrencyLimit: 10, // Limits zlib concurrency for perf.
        threshold: 1024 // Size (in bytes) below which messages
        // should not be compressed if context takeover is disabled.
    }

    const wss = new WebSocketServer({
        port: WS_PORT,
        perMessageDeflate: null // dissable perMessageDeflate
    });
    
    wss.on('connection', function connection(ws) {
        ws.on('error', console.error);
    
        ws.on('message', function message(data) {
            console.log(data);
            console.log(data.length);
            console.log('received: %s', data);
        });
        
        ws.send('something');
    });
}

function startUWS() {

    console.log('start UWS');

    vuws = uws.SSLApp({
        key_file_name: '../certs/key.pem',
        cert_file_name: '../certs/cert.pem'
    }).ws('/*', {
  
        /* There are many common helper features */
        idleTimeout: 200,
        maxBackpressure: 1024,
        maxPayloadLength: 512,
        compression: uws.DEDICATED_COMPRESSOR_8KB,
        upgrade: (res, req, context) => {
            res.upgrade(
               { path: req.getUrl() }, // 1st argument sets which properties to pass to ws object, in this case ip address
               req.getHeader('sec-websocket-key'),
               req.getHeader('sec-websocket-protocol'),
               req.getHeader('sec-websocket-extensions'), // 3 headers are used to setup websocket
               context // also used to setup websocket
            )
        },

        open: (ws)=>{
            console.log("ws open " + " : " + ws.path);
        },
        /* For brevity we skip the other events (upgrade, open, ping, pong, close) */
        message: (ws, message, isBinary) => {
            /* You can do app.publish('sensors/home/temperature', '22C') kind of pub/sub as well */
            
            /* Here we echo the message back, using compression if available */
            let ok = ws.send(message, isBinary, true);
        }
        
    }).listen(WS_PORT, (listenSocket) => {
      
        if (listenSocket) {
          console.log('UWS Listening to port ' + WS_PORT);
        }
        
    });
}

function startWSS() {
    console.log('start WSS');
    let server = null;
    if (0) {
        var options = {
            pfx: readFileSync('../certs/cert.pem'),
            passphrase: 'tomva12a6',
            production : false
        };
        server = createServer(options, (req, res)=>{});
    } else {
        server = createServer({
            key: readFileSync('../certs/key.pem'),
            cert: readFileSync('../certs/cert.pem')
        });
    }
    
    const wss = new WebSocketServer({ server });
    
    wss.on('connection', function connection(ws) {
        ws.on('error', console.error);
        
        ws.on('message', function message(data) {
            console.log('received: %s', data);
        });
        
        ws.send('hello from server');
    });
    
    server.listen(WS_PORT);
}

function main() {
    //startTCP();
    //startWS();
    //startWSS();
    startUWS();
}

main();
