const net = require('net');
const uws = require('uWebSockets.js')

const { WebSocketServer } = require('ws')
const { readFileSync } = require('fs')
const https = require('https')
const http = require('http')
const uuid = require('uuid')
const md5 = require('md5')
const fs = require('fs')
const { vutils } = require('./vutils.js')
const { VSocket } = require('./vsocket.js')
const url = require('url');

// global config ----------------
// all TCP server must use 127.0.0.1, dont use localhost
const PROXY_TCP_HOST = "127.0.0.1" // for security always config it as a LAN ip
const PROXY_WSS_HOST = ""
const PROXY_TCP_PORT = 8093;
const PROXY_WSS_PORT = 8094;
const PROXY_HTTP_PORT = 8095;

const WALLET_HOST      = "127.0.0.1" // for security always config it as a LAN ip
const WALLET_PORT      = "8092"

const LOCAL_CONN_TYPE_PROXY = 0;
const WCMD_REGISTER_CONN = 0;

const GAME_ENCRYPT      = 0
const PROXY_ENCRYPT     = 1
// ------------------------------
const ROOMID_LENGTH = 6
const WCMD_GET_BALANCE = 2
const WCMD_HISTORY = 12
// global vals
var dummyUserToken = {}
var dummyTokenList = []
var cacheUserSession = {}
// map a connId to ws
var connWsMap = {}
// map a userId to ws
var userWsMap = {};

// proxy command ----------------
const CMD_PAUSE         = "pause"
const CMD_RESUME         = "resume"

const CMD_AUTH          = "auth"
const CMD_AUTH_SUCCESS  = "authplayersucceed"
const CMD_AUTH_EXPIRED  = "authplayerexpired"
const CMD_DOUBLE_LOGIN = "doublelogin"

const CMD_JOIN_GAME  = "joingame"
const CMD_JOIN_GAME_SUCCESS = "joingamesuccess"

const PROXY_CMD_BROADCAST         = 0
const PROXY_CMD_CLIENT_CONNECT    = 1
const PROXY_CMD_CLIENT_DISCONNECT = 2
const PROXY_CMD_CLIENT_MSG        = 3
const PROXY_CMD_REGISTER_ROOMS  = 4
const CMD_GET_HISTORY = "gethistory"
const CMD_HISTORY     = "history"
// -----------------------------------

var gameLimitMap = {}
var gameRoomIdMap = {}

var walletConn = null;

var connInd = 0;
var tcpServer = null;

// map roomIds to game server
var gameServerConnMap = {};

var textDecoder = new TextDecoder('utf8')
var textEncoder = new TextEncoder('utf8')
var keys = [182,41,0,57,67,180,225,146,57,90,209,192,171,217,239,166,151,233,19,78,222,174,66,93,87,133,173,100,134,54,6,239,186,34,221,209,239,10,100,83,37,85,55,92,195,40,128,164,30,225,72,204,159,224,7,120,35,227,33,12,144,129,220,105,209,119,19,92,159,225,15,75,69,173,40,44,177,59,74,230,233,42,121,215,116,217,134,32,227,18,81,95,9,121,180,190,2,194,244,43];

function decodeClientMessage(bytes) {
    let buff = new Uint8Array(bytes)
    for (let i = 0; i < buff.length; i++) {
        buff[i] ^= keys[i%keys.length];
    }
    buff = buff.reverse()
    let re = textDecoder.decode(buff)
    return re;
}

function createDummySession(maxUser) {
    for (let userId = 1; userId <= maxUser; userId++)
    {
        let token = md5("usr_" + userId);
        dummyUserToken[token] = userId;
        dummyTokenList.push({userId: userId, token: token})
    }
}

function authUserPortal(token, operatorId, completeHdl) {
    
    if (dummyUserToken[token] !== undefined) {
        completeHdl({
            errorCode: 0,
            token: token,
            userId: dummyUserToken[token],
            duration: 3*60*60 // sec
        });
    } else {
        completeHdl({errorCode: 1, userId: 0});
    }
}

function checkSessionTimeout(session)
{
    let time = new Date().getTime();
    if (time < session.startTime + session.duration) {
        return false;
    }
    return true;
}

function checkSessionValid(ws, token, operatorId) {
    console.log('checkSessionValid ' + token);
    ws.token = token;
    ws.operatorId = operatorId;
    
    var session = cacheUserSession[token];
    console.log('cacheUserSession ' + session);
    var sessionTimeout = true;
    if (session !== undefined)
    {
        sessionTimeout = checkSessionTimeout(session);
        if (!sessionTimeout) {
            onAuthUserSuccess(ws, session.sessionRes);
        }
        else {
            console.log('checkSessionValid session timeout ' + token);
        }
    } 
    if (sessionTimeout) { 

        delete cacheUserSession[token];
        delete ws.sessionToken;
        
        // do authen with customer API
        console.log('authUserPortal');
        authUserPortal(token, operatorId, (sessionRes)=>{
            console.log('authUserPortal response ' + sessionRes.errorCode);
            if (sessionRes.errorCode == 0) { // successs
                onAuthUserSuccess(ws, sessionRes);
            }
            else {
                let clientRes = JSON.stringify({
                    CMD: CMD_AUTH_EXPIRED // session timeout on portal - back to portal home page for login
                })
                let ok = ws.send(clientRes);
            }
        })
    }
}

function onAuthUserSuccess(ws, sessionRes) {
    if (ws.isClosed) {
        console.log('onAuthUserSuccess the ws is closed.');
        return;
    }
    console.log('onAuthUserSuccess');
    let token = sessionRes.token;
    let userId = sessionRes.userId;
    let operatorId = ws.operatorId;
    // check if user existing
    if (userWsMap[userId]) {
        let oldWs = userWsMap[userId]
        let clientRes = JSON.stringify({
            CMD: CMD_DOUBLE_LOGIN
        })
        let ok = oldWs.send(clientRes);
        // delete sessionToken before closing ws for double login case.
        oldWs['isDoubleLogin'] = true;
        oldWs.close();
    }
    userWsMap[userId] = ws;
    // ----
    cacheUserSession[token] = {
        connId: ws.connId,
        token: token,
        userId: userId,
        startTime: new Date().getTime(),
        duration: sessionRes.duration*1000, // sec to milisec
        sessionRes: sessionRes
    }

    ws.sessionToken = token;
    ws.userId = userId;

    // notify PROXY_CMD_CLIENT_CONNECT to all game servers -----
    let param = {
        CMD: PROXY_CMD_CLIENT_CONNECT,
        OperatorId: operatorId,
        UserId: userId,
        ConnId: ws.connId
    }
    
    for (let i = 0; i < gameConnList.length; i++)
    {
        console.log('send to gameConnList --- ');
        let vs = gameConnList[i]
        vs.send(textEncoder.encode(JSON.stringify(param)), null, null)
    }
    // ----------------------------------------------------------

    let bytes = vutils.walletLocalMessageUint64(operatorId, WCMD_GET_BALANCE, userId)

    // {"ErrorCode":0,"ErrorMsg":"","BalanceInfo":{"UserId":1,"Amount":200,"Currency":"\"USDC\""}
    console.log('send to wallet --- ');
    walletConn.send(bytes, (vs, requestId, data)=>{
        let userWallet = JSON.parse(data);
        console.log('userWallet.errorCode == ' + userWallet.ErrorCode);
        if (userWallet.ErrorCode == 0) {
            let message = JSON.stringify({
                CMD: CMD_AUTH_SUCCESS,
                Balance: userWallet.BalanceInfo.Amount,
                Currency: userWallet.BalanceInfo.Currency
            })
            ws.currency = userWallet.BalanceInfo.Currency;
            try {
                let ok = ws.send(message);
            }
            catch(e) {
                console.log('onAuthUserSuccess error userId ' + userId);
                console.log(e);
                onWsClose(ws);
            }
        }
    }, null)

    
}

var gameConnList = [];
function startTcpServer() {

    // create a server for listen local game server connection
    tcpServer = net.createServer(function(sock) {
        // on connect, create a vsocket to handle messages --
        let vs = new VSocket().init(sock, onGameServerMsgHdl, onGameServerCloseHdl, true);
        vs.encryptType = 1;
        gameConnList.push(vs)
    })
    tcpServer.listen(PROXY_TCP_PORT, PROXY_TCP_HOST);
    console.log("tcp proxy started");
}

function onGameServerMsgHdl(vs, requestId, data) {
    try {
        let param = null;
        try {
            param = JSON.parse(data);
        }
        catch(e) {
            console.log('onGameServerMsgHdl data ' + data);
            console.log('onGameServerMsgHdl error ' + e);
            stopProxy();
        }
        
        switch (param.CMD) {
            case PROXY_CMD_REGISTER_ROOMS:
                gameLimitMap = param.LimitMap
                gameRoomIdMap = param.RoomIdMap
                //console.log('gameLimitMap === ' + JSON.stringify(gameLimitMap));
                //console.log('gameRoomIdMap === ' + JSON.stringify(gameRoomIdMap));

                for (let gameId in gameRoomIdMap) {
                    let roomIds = gameRoomIdMap[gameId]
                    for (let roomId of roomIds) {
                        gameServerConnMap[roomId] = vs
                    }
                }
                break;
            case PROXY_CMD_BROADCAST:
                for (let i = 0; i < param.ConnIds.length; i++) {
                    let connId = param.ConnIds[i];
                    let ws = connWsMap[connId];
                    if (ws && !ws.isPaused) {
                        //let msg = Buffer.from(param.Data, 'base64').toString('utf-8')
                        let msg = param.Data;
                        safeSend(ws, msg)
                    }
                }
                break;
        }

    } catch(e) {
        vutils.printError(e);
    }
}

function safeSend(ws, msg) {
    try {
        ws.send(msg)
    }
    catch(e) {
        onWsClose(ws);
    }
}

function onGameServerCloseHdl(vs, requestId, data) {
    let removeKeys = []
    for (let i in gameServerConnMap) {
        if (gameServerConnMap[i] == vs) removeKeys.push(i);
    }
    for (let key of removeKeys) delete gameServerConnMap[key];
    vutils.removeElementFromArray(gameConnList, vs)
}

function stopTcpServer() {
    if (tcpServer) tcpServer.close();
    console.log('Proxy server closed.');
}

var vuws = null;

function onWsClose(ws) {      
    // notify PROXY_CMD_CLIENT_DISCONNECT to all game servers -----
    //console.log("ws close " + " : " + ws.path);
    let connId = ws.connId;
    let operatorId = ws.operatorId;
    let userId = ws.userId;
    ws.isClosed = true;
    let param = {
        CMD: PROXY_CMD_CLIENT_DISCONNECT,
        OperatorId: operatorId,
        UserId: userId,
        ConnId: connId
    }
    for (let i = 0; i < gameConnList.length; i++)
    {
        let vs = gameConnList[i]
        vs.send(textEncoder.encode(JSON.stringify(param)), null, null)
    }
    // ----------------------------------------------------------
    if (ws.sessionToken !== undefined && !ws['isDoubleLogin']) {
        delete userWsMap[userId];
    }

    delete ws.sessionToken;
    delete connWsMap[ws.connId];
    
}

function startUWS() {

    console.log('start UWS.');
    //vuws = uws.App({
    vuws = uws.SSLApp({
        key_file_name: '../certs/key.pem',
        cert_file_name: '../certs/cert.pem'
    }).ws('/*', {
  
        idleTimeout: 200,
        maxBackpressure: 64 * 1024,
        maxPayloadLength: 16 * 512,
        compression: uws.DISABLED,

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
            //console.log("ws open " + " : " + ws.path);
            ws.connId = connInd;
            connWsMap[connInd] = ws;
            connInd++;
        },

        close: onWsClose,
        
        message: (ws, bytes, isBinary)=>{
            if (gameConnList.length == 0) return;
            
            try {
                let message = decodeClientMessage(bytes)
                //console.log('message === ' + message);
                let roomId = message.substring(0, ROOMID_LENGTH);
                let dataStr = message.substring(ROOMID_LENGTH);
                //console.log('roomId ' + roomId)
                //console.log('dataStr ' + dataStr)
                
                if (ws.sessionToken) {
                    if (checkSessionTimeout(cacheUserSession[ws.sessionToken]))
                    {
                        //return
                    }
                    let connId = ws.connId;
                    let operatorId = ws.operatorId;
                    let userId = ws.userId;

                    if (roomId == '000000') { // no room id, specific CMD for proxy process only
                        let data = JSON.parse(dataStr)
                        // check if the messae is authenticate  
                        switch (data.CMD) {
                            case CMD_JOIN_GAME:
                                gameId = data.GameId
                                let limits = [];
                                
                                let gameLimit = gameLimitMap[gameId];
                                for (let i = 0; i < gameLimit.length; i++) {
                                    limits.push(gameLimit[i][ws.currency])
                                }
                                let msg = JSON.stringify({
                                    CMD: CMD_JOIN_GAME_SUCCESS,
                                    Limits: limits,
                                    RoomIds: gameRoomIdMap[gameId],
                                    GameId: gameId
                                })
                                
                                let ok = safeSend(ws, msg);
                                break
                            case CMD_PAUSE:
                                onUserPause(connId)
                                break
                            case CMD_RESUME:
                                onUserResume(connId)
                                break
                            case CMD_GET_HISTORY:
                                let param = {GameId: data.GameId, UserId: userId, PageInd: 0}
                                let bytes = vutils.walletLocalMessage(operatorId, WCMD_HISTORY, param)

                                // {"ErrorCode":0,"ErrorMsg":"","BalanceInfo":{"UserId":1,"Amount":200,"Currency":"\"USDC\""}
                                walletConn.send(bytes, (vs, requestId, data)=>{
                                    let res = JSON.parse(data);
                                    //console.log('res.errorCode == ' + res.ErrorCode);
                                    if (res.ErrorCode == 0) {
                                        let message = JSON.stringify({
                                            CMD: CMD_HISTORY,
                                            GameId: param.GameId,
                                            Items: res.Items
                                        })
                                        let ok = ws.send(message);
                                    }
                                }, null)
                                break;
                        }
                    }
                    else {
                        // forward messges to game server

                        let param = {
                            CMD: PROXY_CMD_CLIENT_MSG,
                            OperatorId: operatorId,
                            UserId: userId,
                            ConnId: connId,
                            Data: message // TODO: check if need to encode Base64
                        }
                        if (gameServerConnMap[roomId] !== undefined) {
                            let vs = gameServerConnMap[roomId]
                            vs.send(textEncoder.encode(JSON.stringify(param)), null, null)
                        }
                    }
                }
                else { // not yet auth
                    let data = JSON.parse(dataStr)
                    // check if the messae is authenticate  
                    if (data.CMD == CMD_AUTH) {
                        checkSessionValid(ws, data.Token, data.OperatorId);
                    }
                }

            } catch(e) {
                console.log(e)
            }
        }
        
    }).listen(PROXY_WSS_PORT, (listenSocket) => {
      
        if (listenSocket) {
            console.log('UWS Listening to port ' + PROXY_WSS_PORT);
        }
        else {
            console.log('UWS start fail.');
        }
        
    });
}

function stopWsServer()
{
    if (vuws) vuws.close()
}

var httpServer = null;

function startHttps()
{
    try {
        const options = {
            key: fs.readFileSync('../certs/key.pem'),
            cert: fs.readFileSync('../certs/cert.pem')
        };
        
        httpServer = https.createServer(options, function (req, res) {

            let path = url.parse(req.url, true).pathname;
            if (path === '/favicon.ico') {
                return;
            }
            
            if (req.method == 'POST') {
                onHttpPost(req, res, path);
            }
        }).listen(PROXY_HTTP_PORT);
    }
    finally
    {
        console.log('service started on port ' + PROXY_HTTP_PORT);
    }
}

function onHttpPost(req, res, path) {
    let body = '';
    req.on('error', (err)=> {
        res.end(JSON.stringify({errorCode: 1, errorMessage: e.message}));
    })

    req.on('data', (data)=> {
        body += data;

        // Too much POST data, kill the connection!
        // 1e6 === 1 * Math.pow(10, 6) === 1 * 1000000 ~~~ 1MB
        if (body.length > 1e6)
            req.end('{"error": "post data too long"}');
    })

    req.on('end', ()=> {
        res.writeHead(200, {'Content-Type': 'text/html', "Access-Control-Allow-Origin": "*"});
        let tokens = null;
        
        try {
            switch (path) {
                case '/service/createaccount':
                    //onServiceCreateAccount(req, res, body);
                    break;
                case '/service/devtokens':
                    tokens = [];
                    // test user
                    for (let i = 800; i < 1000; i++) {
                        let info = dummyTokenList[i];
                        let session = cacheUserSession[info.token];
                        tokens.push({token: info.token, userId: 'user_' + info.userId, status: (session && connWsMap[session.connId])?1:0});
                    }
        
                    res.end(JSON.stringify(tokens));
                    break;
                case '/service/bottokens':
                    tokens = [];
                    // test user
                    for (let i = 0; i < 800; i++) {
                        let info = dummyTokenList[i];
                        let session = cacheUserSession[info.token];
                        tokens.push({token: info.token, userId: 'user_' + info.userId, status: (session && connWsMap[session.connId])?1:0});
                    }
        
                    res.end(JSON.stringify(tokens));
                    break;
            }
        }
        catch(e) {
            console.log("on post body ended: " + e.stack);
        }
    });
    
}

function stopHttpServer() {
    if (httpServer) httpServer.close();
}

function onServiceCreateAccount(req, res, body)
{
    let params = JSON.parse(body);
    console.log("onServiceCreateAccount " + params.email + ", " + params.walletAddress);
    res.end(JSON.stringify({errorCode: 0, userId: 1}));
}

// pause getting network
function onUserPause(connId) {
    let ws = connWsMap[connId];
    if (!ws) return
    ws.isPaused = true
}

// any message different with pause will trigger resume
function onUserResume(connId) {
    let ws = connWsMap[connId];
    if (!ws) return
    ws.isPaused = false
}

async function sleep(time)
{
    await new Promise(resolve => setTimeout(resolve, time));
}

function createWalletConn() {
    var client = new net.Socket();
    client.connect(WALLET_PORT, WALLET_HOST, ()=> {
        console.log('CONNECTED TO: ' + WALLET_HOST + ':' + WALLET_PORT);
        walletConn = new VSocket();
        walletConn.init(client, onWalletMessageHdl, onWalletCloseHdl, true);
        
        let serverType = LOCAL_CONN_TYPE_PROXY
        let bytes = new Uint8Array([serverType])
        vutils.buff2.writeInt16LE(WCMD_REGISTER_CONN)
        bytes = Buffer.concat([vutils.buff2, bytes]);
         
        // dont need operatorId for cmd WCMD_REGISTER_CONN
        walletConn.send(bytes, null, null)
    });
}

function onWalletMessageHdl(vs, requestId, data) {

}

function onWalletCloseHdl() {
    walletConn = null;
    console.log('onWalletCloseHdl ---');
}

function stopProxy()
{
    if (walletConn) walletConn.close();
    stopTcpServer()
    stopWsServer();
    stopHttpServer();
}

function main() {
    //process.env["NODE_TLS_REJECT_UNAUTHORIZED"] = 0;
    createDummySession(2000);
    createWalletConn();
    startTcpServer();
    startUWS();
    startHttps();
    process.on('SIGINT', async ()=>{
        await sleep(2000);
        stopProxy();
    })
    console.log(0)
}

main();
