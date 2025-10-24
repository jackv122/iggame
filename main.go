package main

import (
	"container/list"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"time"
	"unsafe"
	com "vgame/_common"
	wal "vgame/_wallet"
	cock "vgame/cockstrategy"
	roul "vgame/roulette"
)

func test() {
	if false {
		var writeQueryList chan string = make(chan string)
		go func() {
			/*
				for queryStr := range writeQueryList {
					fmt.Println("start query")
					fmt.Println("doing ", queryStr)
					time.Sleep(time.Second * 2)
					fmt.Println("done query")
				}
			*/
			for {
				queryStr := <-writeQueryList
				fmt.Println("start query")
				fmt.Println("doing ", queryStr)
				time.Sleep(time.Second * 20)
				fmt.Println("done query")
			}

		}()
		fmt.Println("assign query 1 --- ")
		writeQueryList <- "query 1"
		fmt.Println("here ---")
		writeQueryList <- "query 2"
		fmt.Println("here ---")
	} else {
		var queryChan chan *string = make(chan *string)
		var writeQueryList = list.New()
		var mutex sync.Mutex
		go func() {
			for queryStr := range queryChan {
				fmt.Println("doing ", *queryStr)
				time.Sleep(time.Second * 3)
				fmt.Println("done query")
			}
		}()
		go func() {
			for {
				var queryStr string = ""
				mutex.Lock()
				if writeQueryList.Len() > 0 {
					queryStr = writeQueryList.Front().Value.(string)
					writeQueryList.Remove(writeQueryList.Front())
				}
				mutex.Unlock()
				fmt.Println("loop here ---")
				if queryStr != "" {
					queryChan <- &queryStr
				}
			}
		}()

		fmt.Println("assign query 1 --- ")
		mutex.Lock()
		writeQueryList.PushBack("query 1")
		writeQueryList.PushBack("query 2")
		mutex.Unlock()
		time.Sleep(time.Second * 2)
		mutex.Lock()
		writeQueryList.PushBack("query 3")
		writeQueryList.PushBack("query 4")
		mutex.Unlock()
		fmt.Println("done assign query --- ")
	}
}

func loadVSettingsFromJSON() {
	configPath := "config/vsettings.json"

	// Check if config file exists, if not use default values
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Config file %s not found, using default values\n", configPath)
		return
	}

	// Read and parse JSON config
	jsonData, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config file: %v, using default values\n", err)
		return
	}

	// Parse JSON into interface{} to access nested values
	var configData map[string]interface{}
	err = json.Unmarshal(jsonData, &configData)
	if err != nil {
		fmt.Printf("Error parsing config file: %v, using default values\n", err)
		return
	}

	// Load operators
	if operators, ok := configData["operators"].(map[string]interface{}); ok {
		com.GlobalSettings.OPERATORS = make(map[com.OperatorID]*com.OperatorInfo)
		for id, opData := range operators {
			if opMap, ok := opData.(map[string]interface{}); ok {
				com.GlobalSettings.OPERATORS[com.OperatorID(id)] = &com.OperatorInfo{
					ID:   com.OperatorID(opMap["id"].(string)),
					Name: opMap["name"].(string),
				}
			}
		}
	}

	// Load database configuration
	if database, ok := configData["database"].(map[string]interface{}); ok {
		if val, exists := database["walletSchema"]; exists {
			com.WALLET_SCHEMA = val.(string)
		}
		if val, exists := database["gameSchema"]; exists {
			com.GAME_SCHEMA = val.(string)
		}
		if val, exists := database["walletUser"]; exists {
			com.WALLET_MYSQL_USER = val.(string)
		}
		if val, exists := database["walletPassword"]; exists {
			com.WALLET_MYSQL_KEY = val.(string)
		}
		if val, exists := database["walletHost"]; exists {
			com.WALLET_MYSQL_HOST = val.(string)
		}
		if val, exists := database["gameUser"]; exists {
			com.GAME_MYSQL_USER = val.(string)
		}
		if val, exists := database["gamePassword"]; exists {
			com.GAME_MYSQL_KEY = val.(string)
		}
		if val, exists := database["gameHost"]; exists {
			com.GAME_MYSQL_HOST = val.(string)
		}
	}

	// Load network configuration
	if network, ok := configData["network"].(map[string]interface{}); ok {
		if val, exists := network["walletHost"]; exists {
			com.WALLET_HOST = val.(string)
		}
		if val, exists := network["walletPort"]; exists {
			com.WALLET_PORT = val.(string)
		}
		if val, exists := network["walletHttpPort"]; exists {
			com.WALLET_HTTP_PORT = val.(string)
		}
		if val, exists := network["proxyTcpHost"]; exists {
			com.PROXY_TCP_HOST = val.(string)
		}
		if val, exists := network["proxyTcpPort"]; exists {
			com.PROXY_TCP_PORT = val.(string)
		}
		if val, exists := network["proxyWssPort"]; exists {
			com.PROXY_WSS_PORT = val.(string)
		}
		if val, exists := network["blockchainUrl"]; exists {
			com.BLOCKCHAIN_URL = val.(string)
		}
	}

	// Load security configuration
	if security, ok := configData["security"].(map[string]interface{}); ok {
		if val, exists := security["hashKey"]; exists {
			com.HASH_KEY = val.(string)
		}
		if val, exists := security["maxAdminConn"]; exists {
			com.MAX_ADMIN_CONN = int(val.(float64))
		}
	}

	// Load game configuration
	if game, ok := configData["game"].(map[string]interface{}); ok {
		if val, exists := game["batchMessage"]; exists {
			com.GAME_BATCH_MESSAGE = val.(bool)
		}
		if val, exists := game["maxAccount"]; exists {
			com.MAX_ACCOUNT = int(val.(float64))
		}
		if val, exists := game["maxRound"]; exists {
			com.MAX_ROUND = int(val.(float64))
		}
		if val, exists := game["operatorIdLength"]; exists {
			com.OPERATOR_ID_LENGTH = int(val.(float64))
		}
		if val, exists := game["maxTrendPageSize"]; exists {
			com.MAX_TREND_PAGE_SIZE = int(val.(float64))
		}
		if val, exists := game["hisPageSize"]; exists {
			com.HIS_PAGE_SIZE = int(val.(float64))
		}
		if val, exists := game["vsocketSendTimeout"]; exists {
			if duration, err := time.ParseDuration(val.(string)); err == nil {
				com.VSOCKET_SEND_TIMEOUT = duration
			}
		}
	}

	// Load tables configuration
	if tables, ok := configData["tables"].(map[string]interface{}); ok {
		if val, exists := tables["walletTable"]; exists {
			com.WALLET_TABLE = val.(string)
		}
	}

	fmt.Println("VSettings loaded from JSON configuration")
}

func initGameFactory() {

	com.GameFactory = func(gameId com.GameId, gameServer *com.GameServer) com.GameInterface {
		var game com.GameInterface = nil
		switch gameId {
		case com.IDGameA:
			game = (&com.GameA{}).Init(gameServer)
			// type assertion of Interface
			gameServer.SetGamePointer(gameId, unsafe.Pointer(game.(*com.GameA)))
		case com.IDRoulette:
			game = (&roul.Roulette{}).Init(gameServer)
			gameServer.SetGamePointer(gameId, unsafe.Pointer(game.(*roul.Roulette)))
		case com.IDCockStrategy:
			game = (&cock.CockStrategy{}).Init(gameServer)
			gameServer.SetGamePointer(gameId, unsafe.Pointer(game.(*cock.CockStrategy)))
		}
		return game
	}

	com.GetGameInterface = func(gameId com.GameId, gameServer *com.GameServer) com.GameInterface {
		var game com.GameInterface = nil
		switch gameId {
		case com.IDRoulette:
		case com.IDRoulette_02: // another instance of roulette game. We can have many game instances to handle different room configs
			game = (*roul.Roulette)(gameServer.GetGamePointer(gameId))
		case com.IDGameA:
			game = (*com.GameA)(gameServer.GetGamePointer(gameId))
		case com.IDCockStrategy:
			game = (*cock.CockStrategy)(gameServer.GetGamePointer(gameId))
		}
		return game
	}

}

func main() {
	loadVSettingsFromJSON()
	initGameFactory()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	if len(os.Args) > 1 && os.Args[1] == "--createacc" {
		fmt.Println("createacc ...")
		com.TestWalletTcpClient.CreateTestAcc()
	} else if len(os.Args) > 1 && os.Args[1] == "--addbalance" {
		fmt.Println("addbalance ...")
		com.TestWalletTcpClient.AddBalance()
	} else if len(os.Args) > 1 && os.Args[1] == "--test" {
		//time.Sleep(3 * time.Second)
		//com.TestVSocket.DoTest()
		//com.TestWalletHttpClient.DoTest()
		com.TestWalletTcpClient.DoTest()
		//com.TestGameTcpClient.DoTest()
		/*
			cd /Users/fe-tom/Documents/private_pros/vgame/research/golang
			go run main.go --test
		*/
		fmt.Println("hi --")
		fmt.Println(0)
		//test()
		<-sigs

		fmt.Println("exit test")

	} else {
		var serverName = flag.String("s", "game", "server name: game, wallet")
		flag.Parse()
		switch {
		case *serverName == "wallet":
			fmt.Println("start Wallet")
			// start all operators
			wal.Wallet.Start(nil)
			fmt.Println("Wallet started...")
			fmt.Println(0)
			<-sigs // first wait for ctrl + c
			time.Sleep(time.Second * 4.0)
			wal.Wallet.Stop()

		case *serverName == "game":
			fmt.Println("start Game")
			// start all operators
			com.Game.Start()
			fmt.Println("Game started...")
			fmt.Println(0)
			<-sigs // first wait for ctrl + c
			com.Game.Maintenance()
		}

		//vtest.TestHelloWorld()
		//vtest.TestMultithread()
		//vtest.TestServer()

		// for multiple server starting
		// go vtest.StartSocket()
	}

}
