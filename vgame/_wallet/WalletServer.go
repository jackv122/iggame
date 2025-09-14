package wal

import (
	"fmt"
	"net"
	"os"
	"unsafe"
	com "vgame/_common"

	_ "github.com/go-sql-driver/mysql"
)

type CommonBalanceParam struct {
	Token    string
	Infos    []*com.AmountInfo
	Checksum uint64
}

type WalletServer struct {
	operatorList []com.OperatorID
	listener     net.Listener
	operatorMap  map[com.OperatorID]*WalletServerOperator

	proxyConns []*com.VSocket
	gameConns  []*com.VSocket

	isMaintenance bool
}

var Wallet = (&WalletServer{}).Init()

func (s *WalletServer) Init() *WalletServer {
	s.operatorMap = map[com.OperatorID]*WalletServerOperator{}
	for operatorId, _ := range com.GlobalSettings.OPERATORS {
		s.operatorMap[operatorId] = (&WalletServerOperator{}).Init()
	}

	s.proxyConns = []*com.VSocket{}
	s.gameConns = []*com.VSocket{}

	return s
}

func (s *WalletServer) Start(operators []com.OperatorID) {
	if operators == nil {
		operators = []com.OperatorID{}
		for operatorId := range s.operatorMap {
			operators = append(operators, operatorId)
		}
	}
	s.operatorList = operators
	for _, operatorId := range operators {
		operator := s.operatorMap[operatorId]
		operator.Start(operatorId)
	}
	//s.StartHTTPServer()
	go s.startSocketServer()
}

func (s *WalletServer) startSocketServer() {
	listener, err := net.Listen("tcp", com.WALLET_HOST+":"+com.WALLET_PORT)
	if err != nil {
		fmt.Println("Wallet start error:", err.Error())
		os.Exit(1)
	}
	s.listener = listener
	defer func() {
		fmt.Println("Wallet server closed.")
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			//fmt.Println("Error accepting: ", err.Error())
			return
		}
		go func() {
			(&com.VSocket{}).Init(conn, s.onMsgHdl, s.onConnCloseHdl, true, 4, 1)
		}()
	}
}

// tcp message: [msglen - 2bytes][ck - 8bytes][isResponse - 1byte][requestId - 8bytes][operatorId - 3bytes][cmd - 2bytes][json]
func (s *WalletServer) onMsgHdl(vs *com.VSocket, requestId uint64, data []byte) {
	if vs.UserData == nil {
		cmd := com.VUtils.BytesToUint16(&data)
		if cmd == com.WCMD_REGISTER_CONN {
			conInfo := com.LocalConnInfo{}
			conInfo.ConnType = data[2]
			vs.UserData = unsafe.Pointer(&conInfo)
			switch conInfo.ConnType {
			case com.LOCAL_CONN_TYPE_GAME:
				// set encryptType for later response encrypt
				vs.EncryptType = 0 // game encrypt type
				s.gameConns = append(s.gameConns, vs)
			case com.LOCAL_CONN_TYPE_PROXY:
				// set encryptType for later response encrypt
				vs.EncryptType = 1 // proxy encrypt type
				s.proxyConns = append(s.proxyConns, vs)
			}
			res := "success"
			vs.Response(requestId, []byte(res))
		}

		return
	}

	conInfo := (*com.LocalConnInfo)(vs.UserData)

	//fmt.Println("onMsgHdl - ", conInfo.ConnType)
	if conInfo.ConnType == com.LOCAL_CONN_TYPE_PROXY {
		s.onProxyMsgHdl(vs, requestId, data)
	} else if conInfo.ConnType == com.LOCAL_CONN_TYPE_GAME {
		s.onGameMsgHdl(vs, requestId, data)
	}

}

func (s *WalletServer) onProxyMsgHdl(vs *com.VSocket, requestId uint64, data []byte) {
	operatorId := com.OperatorID(data[:com.OPERATOR_ID_LENGTH])
	opServer, operatorExist := s.operatorMap[operatorId]
	if !operatorExist {
		return
	}
	opServer.onProxyMsgHdl(vs, requestId, data[com.OPERATOR_ID_LENGTH:])
}

func (s *WalletServer) onGameMsgHdl(vs *com.VSocket, requestId uint64, data []byte) {
	operatorId := com.OperatorID(data[:com.OPERATOR_ID_LENGTH])
	opServer, operatorExist := s.operatorMap[operatorId]
	if !operatorExist {
		return
	}
	opServer.onGameMsgHdl(vs, requestId, data[com.OPERATOR_ID_LENGTH:])
}

func (s *WalletServer) onConnCloseHdl(vs *com.VSocket) {
}

func (s *WalletServer) CloseAllConnections() {
	for _, conn := range s.gameConns {
		conn.Close()
	}
	for _, conn := range s.proxyConns {
		conn.Close()
	}
}

func (s *WalletServer) Stop() {
	for _, operator := range s.operatorMap {
		operator.Stop()
	}

	s.listener.Close()
}

func (s *WalletServer) sendMaintenance(conn *com.VSocket) {
	bytes := com.VUtils.Uint64ToBytes(com.WCMD_MAINTENANCE)
	conn.Send(bytes, nil, nil)
}

func (s *WalletServer) maintenance() {
	if s.isMaintenance {
		return
	}
	s.isMaintenance = true
	for _, operator := range s.operatorMap {
		operator.maintenance()
	}

	for _, conn := range s.gameConns {
		s.sendMaintenance(conn)
	}
	for _, conn := range s.proxyConns {
		s.sendMaintenance(conn)
	}
}
