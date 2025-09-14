const net = require('net');
const WebSocket = require('ws')

var HOST = 'localhost';
var TCP_PORT = 8093;
var WS_PORT = 8094;

function testClientSocket() {
    var client = new net.Socket();
    client.connect(TCP_PORT, HOST, function() {
        console.log('CONNECTED TO: ' + HOST + ':' + TCP_PORT + '/server01');
        // Write a message to the socket as soon as the client is connected, the server will receive it as message from the client
        //client.write('I am Chuck Norris! --- ');
        let arr = new Uint8Array(2);
        arr[0] = 123;
        arr[1] = 255;
        client.write(arr);
    });

    // Add a 'data' event handler for the client socket
    // data is what the server sent to this socket
    client.on('data', function(data) {
        console.log('DATA: ' + data);
        // Close the client socket completely
        client.destroy();
    });

    // Add a 'close' event handler for the client socket
    client.on('close', function() {
        console.log('Connection closed');
    });
}

function keepAlive(ws) {
    setInterval(()=>{
        ws.send(JSON.stringify(msg));
    }, 50000);
}

function testWs() {
    console.log('test ws wss://vgws.vntrend.org:8094');
    const ws = new WebSocket('wss://vgws.vntrend.org:8094', {
        perMessageDeflate: false // disable compression
    });

    ws.on('error', console.error);

    ws.on('open', function open() {
        console.log('ws open -- ');
        //ws.send('aaa');
        ws.close();
    });

    ws.on('message', function message(data) {
        //console.log('received: %s', data);
    });

    ws.on('close', function message() {
        console.log('closed');
    });
    
}

function testInt() {
    var buf = Buffer.alloc(4);
    
    buf.writeUInt16BE(0x1234, 0);
    console.log("buff ", buf)
    var num = buf.readUInt16BE(0);
    console.log("num 0x", num.toString(16))

    buf.reverse(); // convert to big endian
    console.log("buff ", buf)
}

function testInt64() {
    var buf = Buffer.alloc(8);
    
    buf.writeBigInt64LE(BigInt(0x1234), 0);
    console.log("buff ", buf)
    var num = buf.readBigInt64LE(0);
    console.log("num 0x", num.toString(16))
    console.log("buff ", buf)
    buf.copy(buf, 0, 1)
    console.log("buff after fill ", buf)
}

function main() {

    process.env["NODE_TLS_REJECT_UNAUTHORIZED"] = 0;

    //testClientSocket();
    testWs();

    //testInt64();
}

main();

