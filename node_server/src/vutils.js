var vutils = {}

vutils.buff2 = Buffer.alloc(2);
vutils.buff4 = Buffer.alloc(4);
vutils.buff8 = Buffer.alloc(8);
vutils.buff9 = Buffer.alloc(9);
vutils.buff17 = Buffer.alloc(17);

vutils.removeElementFromArray = function(arr, e)
{
    for ( var i = 0; i < arr.length; i++ )
    {
        if (arr[i] === e)
        {
            arr.splice(i, 1);
            return;
        }
    }
}
vutils.textEncoder = new TextEncoder('utf-8')
vutils.walletLocalMessage = function(operatorId, cmd, obj)
{
    let bytes = vutils.textEncoder.encode(JSON.stringify(obj));
    vutils.buff2.writeInt16LE(cmd)
    bytes = Buffer.concat([vutils.buff2, bytes]);
    bytes = Buffer.concat([vutils.textEncoder.encode(operatorId), bytes]);
    return bytes
}

vutils.walletLocalMessageUint64 = function(operatorId, cmd, n)
{
    let bytes = vutils.buff8
    bytes.writeBigInt64LE(BigInt(n));
    vutils.buff2.writeInt16LE(cmd)
    bytes = Buffer.concat([vutils.buff2, bytes]);
    bytes = Buffer.concat([vutils.textEncoder.encode(operatorId), bytes]);
    return bytes
}

vutils.printError = function(e) {
    console.log('error: ' + e.stack)
}

module.exports = { vutils }