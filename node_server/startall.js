const { spawn, execSync } = require("child_process");
function execute(cmd)
{
    console.log('execute: ' + cmd);
    let r;
    let arr
    arr = execSync(cmd, {encoding: 'utf8'}).split('\n');
    r = arr[arr.length - 2];
    if (r == '0')
    {
        console.log('DONE');
        return true;
    }
    console.log('FAIL');
    return false;
}

function vspaw(cmd, params, completHdl)
{
    var cmdStr = cmd + ' ' + params.join(' ')
    console.log('execute: ' + cmdStr);
    let child = spawn(cmd, params);
    child.stdout.on('data', (data) => {
        let msg = `${data}`
        console.log(msg);
        arr = msg.split('\n');
        r = arr[arr.length - 2];
        if (r == '0')
        {
            console.log('DONE');
            completHdl && completHdl()
        }
    });
    child.on('exit', function (code, signal) {
        //console.log('"' + cmdStr + '"' + ' exited with ' + `code ${code} and signal ${signal}`);
    });
    return child
}
let wallet = null;
let proxy = null;
let game = null;

let startWallet = (completeHdl)=>{
    console.log("startWallet --- ");
    wallet = vspaw('go', ['run', '../main.go', '-s', 'wallet'], completeHdl);
}

let startProxy = (completeHdl)=>{
    console.log("startProxy --- ");
    proxy = vspaw('node', ['src/proxy.js'], completeHdl);
}

let startGame = (completeHdl)=>{
    console.log("startGame --- ");
    game = vspaw('go', ['run', '../main.go', '-s', 'game'], completeHdl);
}

async function sleep(time)
{
    await new Promise(resolve => setTimeout(resolve, time));
}

process.stdin.on('data', data => {
    let input = `${data}`.trim()
    if (input == 'exit') {
        stopAll();
    }
});

function killPort(port) {
    try {
        let result = execSync('netstat -vanp tcp | grep ' + port, {encoding: 'utf8'});
        let arr = result.split(/\s+/)
        let pId = arr[8]
        result = execSync('kill -9 ' + pId, {encoding: 'utf8'});
        console.log('kill port ' + port)
    }
    catch(e) {
    }
}

function killPorts()
{
    killPort(WALLET_PORT)
    killPort(PROXY_TCP_PORT)
    killPort(PROXY_WSS_PORT)
}

async function stopAll()
{
    console.log('start exit all servers ------------------');
    await sleep(5000)
    console.log('done exit all servers ------------------');
    //game?.kill();
    //proxy?.kill();
    //wallet?.kill();
    process.exit()
}

process.on('SIGINT', stopAll)

let startAll = ()=>{
    killPorts();
    startWallet(startProxy.bind(this, startGame));
}

const WALLET_PORT      = "8092"
const PROXY_TCP_PORT   = "8093"
const PROXY_WSS_PORT   = "8094"
// lsof -i :8092 
// netstat -vanp tcp | grep 8092
// kill -9 <PID>
//pm2 start 0 --kill-timeout 6000

// cd /Users/fe-tom/Documents/private_pros/vgame/research/golang

startAll()

