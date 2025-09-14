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
    child.stderr.on('data', function(data) {
        console.log('stderr: ' + data);
    });
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

//vspaw('../../research/golang/bin/main', ['-s', 'wallet'], null);
vspaw('./bin/main', ['-s', 'game'], null);
async function sleep(time)
{
    await new Promise(resolve => setTimeout(resolve, time));
}
