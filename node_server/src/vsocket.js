const { vutils } = require("./vutils");

class VSocketCallback {
	
    constructor() {
        this.succesHdl = null;  //func(vs *VSocket, resBytes []byte)
	    this.timeoutId = 0; // timeoutId
    }
}

class VSocket {
    constructor() {
        this.onCloseHdl = null; //func(vs *VSocket)
        this.onMsgHdl = null; //func(vs *VSocket, requestId uint64, resBytes []byte)
        this.checksum = 0;
        this.receiveChecksum = 0;
        // it is interface, dont user pointer to it
        this.conn = null;
        this.responseHdlMap = {}; //map[uint64]*VSocketCallback
        this.msgBytes = null;
        this.contentSize = 0;
        this.closed   = false;
        this.syncCk   = false;
        this.isValid  = false;
        this.encoding = 'utf8';
        this.encryptType = 1;
        this.userData = null;
    }

    /**
     * 
     * @param {*} conn 
     * @param {*} onMsgHdl // func(vs *VSocket, requestId uint64, resBytes string)
     * @param {*} func // func(vs *VSocket)
     */
    init(conn, onMsgHdl, closeHdl, syncCk) {
        this.encrypted = true
        this.onMsgHdl = onMsgHdl
        this.onCloseHdl = closeHdl
        this.checksum = 0
        this.receiveChecksum = 0
        this.contentSize = 0
        this.syncCk = syncCk
        this.msgBytes = Buffer.alloc(0);
        this.isValid = true
        this.responseHdlMap = {}
        this.conn = conn
        
        conn.on('data', (data)=> {
            try {
                this.msgBytes = Buffer.concat([this.msgBytes, data]);
                this.loopParseMsg();
            } catch(e) {
                console.log(e);
            }
        })

        conn.on('close', ()=> {
            //console.log('on conn closed');
            if (this.onCloseHdl) this.onCloseHdl();
        })  

        conn.on('error', (e)=> {
            //console.log(e);
        })

        return this
    }

    encrypt(bytes) 
    {
    }

    decrypt(bytes) 
    {
        // decrypt packageBytes bytes[3:]
        
        // ---
    }
    
    // [msglen - 2bytes][encryptType - 1byte][ck - 8bytes][isResponse - 1byte][requestId - 8bytes][operatorId - 3bytes][cmd - 2bytes][json]
    loopParseMsg() {
        // first 2 bytes is uint16 represent the package size
        let uint32Size = 4
        let msglen = this.msgBytes.length
        if (msglen > uint32Size) {
            this.contentSize = this.msgBytes.readUInt32LE(0);
        }
    
        while (this.contentSize > 0 && msglen >= uint32Size + this.contentSize) {
            let dec = new TextDecoder('utf-8')
            let encryptType = this.msgBytes[uint32Size];
            this.decrypt(this.msgBytes);
            
            let startInd = uint32Size + 1;
            let receiveCK = this.msgBytes.readBigInt64LE(startInd);
            
            if (this.syncCk && this.receiveChecksum != receiveCK) {
                vutils.printError(new Error("duplicate checksum. this.receiveChecksum " + this.receiveChecksum + " receiveCK " + receiveCK))
                this.isValid = false
                return
            }
            let isResponse = this.msgBytes[startInd + 8] > 0
            let requestId = this.msgBytes.readBigInt64LE(startInd + 9);
            let data = this.msgBytes.toString(this.encoding, startInd + 17, uint32Size + this.contentSize);
            
            // it is a callback's response
            if (isResponse) {
                let cb = this.responseHdlMap[requestId]
    
                if (cb) {
                    if (cb.timeoutId != null) {
                        clearTimeout(cb.timeoutId);
                    }
                    if (cb.succesHdl != null) {
                        cb.succesHdl(this, requestId, data)
                    }
    
                    delete this.responseHdlMap[requestId]
                }
            } else { // it is a normal incoming message
                if (this.onMsgHdl != null) {
                    this.onMsgHdl(this, requestId, data)
                }
            }
    
            // reset vs ---
            let temp = Buffer.alloc(this.msgBytes.length - (uint32Size + this.contentSize))
            this.msgBytes.copy(temp, 0, uint32Size + this.contentSize);
            this.msgBytes = temp;
            // update receiveChecksum
            this.receiveChecksum++
            msglen = this.msgBytes.length
    
            if (msglen > uint32Size) {
                this.contentSize = this.msgBytes.readUInt32LE(0);
            } else {
                this.contentSize = 0
            }
            // ------------
        }
    }
    
    /**
    A--- send() ---> B
    if B determine the message need a response by GAME LOGIC (not by protocol) ==> B --- response() ---> A, otherwise B do nothing
    So: when A use send(), only sepecific a timeoutHdl() IF A make sure the message will be response() by B
    */
    /**
     * 
     * @param {*} bytes 
     * @param {*} succesHdl //func(vs *VSocket, resData string)
     * @param {*} timeoutHdl 
     */
    send(msg, succesHdl = null, timeoutHdl = null) {
        let bytes = Buffer.from(msg, this.encoding);
        if (this.closed) {
            return new Error("socket is closed");
        }
        
        // IMPORTANT: we use send checksum as requestid for callback determine
        let requestId = BigInt(this.checksum);
    
        let callback = new VSocketCallback();
        callback.succesHdl = succesHdl;
        this.responseHdlMap[requestId] = callback;
    
        let bt = vutils.buff17; 
        bt.writeBigInt64LE(BigInt(this.checksum), 0);
        bt[8] = 0;
        bt.writeBigInt64LE(BigInt(this.checksum), 9);
        
        bytes = Buffer.concat([bt, bytes]);
    
        this.encrypt(bytes)
        bytes = Buffer.concat([new Uint8Array([this.encryptType]), bytes]);

        let l = bytes.length;
        vutils.buff4.writeUInt32LE(l);

        bytes = Buffer.concat([vutils.buff4, bytes]);

        this.conn.write(bytes);

        if (timeoutHdl != null) {
            // capture this.checksum
            callback.timeoutId = setTimeout(()=>{
                timeoutHdl()
                delete this.responseHdlMap[requestId];
            }, 20 * 1000);
        }
    
        this.checksum++
        return null;
    }
    
    response(requestId, msg) {
        let bytes = Buffer.from(msg, this.encoding);
        if (this.closed) {
            return new Error("socket is closed");
        }
    
        let bt = vutils.buff17; 
        bt.writeBigInt64LE(BigInt(this.checksum), 0);
        bt[8] = 1;
        bt.writeBigInt64LE(BigInt(requestId), 9);
        
        bytes = Buffer.concat([bt, bytes]);

        this.encrypt(bytes)
        bytes = Buffer.concat([new Uint8Array([this.encryptType]), bytes]);

        let l = bytes.length;
        vutils.buff4.writeUInt32LE(l);
        bytes = Buffer.concat([vutils.buff4, bytes]);

        this.conn.write(bytes);
        return null
    }
    
    close() {
        if (this.conn != null) {
            this.conn.destroy()
            this.conn = null
        }
    }
	
}

module.exports = { VSocket }

