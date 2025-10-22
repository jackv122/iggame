
var version = '1.0.0'
// params ---
var DES = 'pair_01'
var LEFT = 'c01'
var RIGHT = 'c02'
// ---------
var logEnable = true;

(function() {

    var config = {
        c01: {
            id: 'c01',
            name: 'Thunder',
            s: 90,
            a: 54,   
        },
        c02: {
            id: 'c02',
            name: 'Storm',
            s: 70,
            a: 86,
        },
        cock3: {
            id: 'c03',
            name: 'Storm',
            s: 62,
            a: 82,
        },
        strength_skills: [
            {
                name: 'neck_kick',
                attack: {
                    name: 'neck_kick_A',
                    damage: 15,
                    cooldown: 5,
                    dur: 1.833,
                    buff: 0.3,
                    stamina: 5
                },
                defend: {
                    name: 'neck_kick_D',
                    damage: 0,
                    cooldown: 0,
                    dur: 1.833,
                    buff: 0,
                    stamina: 10
                }
            }
        ],
        agi_skills: [
            {
                name: 'both_kick',
                attack: {
                    name: 'both_kick_A',
                    damage: 10,
                    cooldown: 5,
                    dur: 1.833,
                    buff: 0.5,
                    stamina: 3
                },
                defend: {
                    name: 'both_kick_D',
                    damage: 4,
                    cooldown: 0,
                    dur: 1.833,
                    buff: 0.2,
                    stamina: 7
                }
            }
        ],
        max_buff_damage: 5
    }

    function vlog(...args) {
        if (!logEnable) return
        const now = new Date();

        const minutes = now.getMinutes();        
        const seconds = now.getSeconds();       
        const milliseconds = now.getMilliseconds()
        console.log(`${minutes}:${seconds}${Math.floor(milliseconds/10)}:`, ...args)
    }

    class CockState {
        static IDLE = 'idle'
        static RUNNING_SKILL = 'running_skill'
        static DIE = 'die'
        static WIN = 'win'
    }

    class Cock {
        id = ''
        name = ''
        s = 0
        a = 0
        stamina = 0
        blood = 0
        fullBlood = 0
        coolDownMap = {}
        state
        runningSkill = null
        skillTime = 0

        checkActiveSkillTime = 0

        onChangeStateHdl = null
        onChangeBloodHdl = null

        enemy = null
        MIN_STAMINA = 50

        game = null

        init(conf, enemy, game, onChangeStateHdl, onChangeBloodHdl) {
            this.game = game
            this.name = conf.name
            this.id = conf.id
            this.enemy = enemy
            this.onChangeStateHdl = onChangeStateHdl
            this.onChangeBloodHdl = onChangeBloodHdl
            this.a = conf.a
            this.s = conf.s
            this.fullBlood = this.s*0.7 + this.a*0.3
            
            this.blood = this.fullBlood
            this.state = CockState.IDLE
            
            this.checkActiveSkillTime = 1.0

            this.stamina = this.s*0.6 + this.a*0.4

            vlog(this.name, 'fullBlood: ', this.fullBlood, 'stamina', this.stamina)
        }

        playSkill(skill)
        {
            vlog(this.name, ' playSkill ', skill.name, 'stamina', this.stamina)
            this.runningSkill = skill
            this.skillTime = skill.dur
            this.stamina -= skill.stamina
            this.coolDownMap[skill.name] = skill.cooldown
            this.setState(CockState.RUNNING_SKILL, skill.name)
        }

        onDamage(dam, stamina)
        {
            this.blood -= dam
            vlog(this.name, 'onDamage', dam,  'blood ', this.blood)
            this.onChangeBloodHdl && this.onChangeBloodHdl(this.blood)
            if (this.blood <= 0)
            {
                this.blood = 0
                this.setState(CockState.DIE)
                this.enemy.setState(CockState.WIN)
            }
        }

        setState(state, data = null) {
            vlog(this.name, 'setState', state)
            this.state = state
            this.onChangeStateHdl && this.onChangeStateHdl(state, data)
            if (state == CockState.WIN) {
                this.game.endGame(this.id)
            }
        }

        update(dt) {
            // decrease 3 stamina / 1s
            this.stamina -= dt*2
            if (this.stamina < this.MIN_STAMINA) this.stamina = this.MIN_STAMINA
            switch(this.state) {
                case CockState.IDLE:
                    for (let i in this.coolDownMap)
                    {
                        this.coolDownMap[i] -= dt
                    }
                    break
                case CockState.RUNNING_SKILL:
                    this.skillTime -= dt
                    if (this.skillTime < 0) {
                        
                        let skill = this.runningSkill
                        let damage = skill.damage + skill.damage*skill.buff*this.game.random()
                        this.enemy.onDamage(damage)

                        this.setState(CockState.IDLE)
                    }
                    break
                case CockState.DIE:
                    break
                case CockState.WIN:
                    break
            }
        }

        checkActiveSkill(dt) 
        {
            this.checkActiveSkillTime -= dt;
            if (this.checkActiveSkillTime < 0) {
                this.checkActiveSkillTime = 1.0
                // running check
                let r = this.game.random()*this.stamina
                if (r > this.MIN_STAMINA*0.7) {
                    let range = this.s + this.a
                    let r = this.game.random() * range
                    let skill = null
                    let skills = null
                    if (r < this.s) {
                        skills = config.strength_skills
                    }
                    else skills = config.agi_skills
                    let ind = Math.floor(this.game.random() * (skills.length - 0.0001))
                    skill = skills[ind]
                    if (this.coolDownMap[skill.name] === undefined || this.coolDownMap[skill.name] <= 0)
                    {
                        // play skill
                        this.playSkill(skill.attack)
                        this.enemy.playSkill(skill.defend)
                    }
                }
                
            }
        }
    }

    class Engine {
        gameData = null
        isReplay = false
        cock1 = null
        cock2 = null
        cock1ChangeStateHdl = null
        cock1ChangeBloodHdl = null

        cock2ChangeStateHdl = null
        cock2ChangeBloodHdl = null

        onEndGame = null

        gameRunning = false
        
        onEndGameHdl = null
        frameTime = 1.0/30
        randoms = []
        gameDur = 0

        constructor() {
            this.cock1 = new Cock()
            this.cock2 = new Cock()   
        }

        startGame(leftCockConfig, rightCockConfig, isReplay = false, gameData = {}, onEndGame = null)
        {
            this.gameData = gameData
            this.onEndGame = onEndGame
            if (!isReplay) this.gameData.randoms = []
            this.gameRunning = true
            this.gameDur = 0
            this.cock1.init(leftCockConfig, this.cock2, this, this.cock1ChangeStateHdl, this.cock1ChangeBloodHdl)
            this.cock2.init(rightCockConfig, this.cock1, this, this.cock2ChangeStateHdl, this.cock2ChangeBloodHdl)
        }

        endGame(winner)
        {
            vlog('winner', winner)
            this.gameRunning = false
            vlog('winner', winner, 'gameDur', (this.gameDur).toFixed(2))
            this.gameData.winner = winner
            this.gameData.duration = this.gameDur
            this.onEndGame && this.onEndGame(this.gameData)
        }

        random()
        {
            if (!this.isReplay) {
                let r = Math.random().toFixed(3)
                if (this.gameData.randoms === undefined) this.gameData.randoms = []
                this.gameData.randoms.push(r)
                return parseFloat(r)
            }
            else {
                let r = this.gameData.randoms.shift()
                return parseFloat(r)
            }
        }

        update() {
            if (!this.gameRunning) return
            this.gameDur += this.frameTime
            this.cock1.update(this.frameTime)
            this.cock2.update(this.frameTime)
            // check running skill if both cocks idle
            if (this.cock1.state == CockState.IDLE && this.cock2.state == CockState.IDLE)
            {
                if (!this.cock1.checkActiveSkill(this.frameTime)) this.cock2.checkActiveSkill(this.frameTime)
            }
        }
    }

    const isBrowser = typeof window !== 'undefined';
    if (isBrowser) {
        window['vlib'] = {}
        window['vlib']['v' + version] = {}
        let v = window['vlib']['v' + version]
        v['Engine'] = Engine
    }

    function main() {
        let engine = new Engine()
        
        let stats = {}

        stats.version = version
        
        stats.leftCockConfig = config[LEFT]
        stats.rightCockConfig = config[RIGHT]
        
        stats.total = 10000
        stats.win = {}
        stats.win[LEFT] = 0
        stats.win[RIGHT] = 0
        stats.minDur = 100000000
        stats.maxDur = 0
        stats.fullWin = {}
        stats.fullWin[LEFT] = 0
        stats.fullWin[RIGHT] = 0
        stats.leftCockConfig = config[LEFT]
        stats.rightCockConfig = config[RIGHT]
        let db = []
        
        if (1) { // gen game datas
            logEnable = false
            for (let i = 0; i < stats.total; i++)
            {
                engine.startGame(config[LEFT], config[RIGHT]);
                for (let j = 0; j < 30*100; j++) if (engine.gameRunning) {
                    engine.update()
                }
                if (engine.gameRunning) throw new Error('game not finish')
                engine.gameData.index = i;

                stats.win[engine.gameData.winner]++
                let isFullWin = false
                if (engine.cock1.blood == engine.cock1.fullBlood) {
                    stats.fullWin[engine.cock1.id]++
                    isFullWin = true
                }
                else if (engine.cock2.blood == engine.cock2.fullBlood) {
                    stats.fullWin[engine.cock2.id]++
                    isFullWin = true
                }
                engine.gameData.isFullWin = isFullWin

                db.push(JSON.stringify(engine.gameData))

                if (stats.minDur > engine.gameDur) stats.minDur = engine.gameDur
                if (stats.maxDur < engine.gameDur) stats.maxDur = engine.gameDur
            }

            logEnable = true
            console.log('done', JSON.stringify(stats))

            // save to config
            const fs = require('fs')
            const path = require('path')
            const statsDir = `../../../config/cock_strategy/${version}/` + DES
            if (!fs.existsSync(statsDir)) {
                fs.mkdirSync(statsDir, { recursive: true })
            }
            fs.writeFileSync(path.join(statsDir, 'stats.json'), JSON.stringify(stats, null, 2))
            console.log(`Stats saved to ${DES}/stats.json`)

            fs.writeFileSync(path.join(statsDir, `db.txt`), db.join('\n'))
            console.log(`db saved to ${DES}/db.txt`)
            
        }
        else {
            setInterval(()=>{
                engine.update()
            }, 1000/30)
            engine.startGame(config[LEFT], config[RIGHT])
        }
    }

    if (!isBrowser) main()
})()
