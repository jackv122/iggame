package main

import (
	"container/list"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"
	"unsafe"
	com "vgame/_common"
	wal "vgame/_wallet"
	rou "vgame/roulette"
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

func initGameFactory() {

	com.GameFactory = func(gameId com.GameId, gameServer *com.GameServer) com.GameInterface {
		var game com.GameInterface = nil
		switch gameId {
		case com.IDGameA:
			game = (&com.GameA{}).Init(gameServer)
			// type assertion of Interface
			gameServer.SetGamePointer(gameId, unsafe.Pointer(game.(*com.GameA)))
		case com.IDRoulette:
			game = (&rou.Roulette{}).Init(gameServer)
			gameServer.SetGamePointer(gameId, unsafe.Pointer(game.(*rou.Roulette)))
		}
		return game
	}

	com.GetGameInterface = func(gameId com.GameId, gameServer *com.GameServer) com.GameInterface {
		var game com.GameInterface = nil
		switch gameId {
		case com.IDRoulette:
			game = (*rou.Roulette)(gameServer.GetGamePointer(gameId))
		case com.IDGameA:
			game = (*com.GameA)(gameServer.GetGamePointer(gameId))
		}
		return game
	}

}

func main() {
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
