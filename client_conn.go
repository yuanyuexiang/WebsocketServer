package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/gorilla/websocket"
	// "log"
	"net/http"
	"strings"
	"time"
)

var (
	writeWait      time.Duration
	pongWait       time.Duration
	pingPeriod     time.Duration
	maxMessageSize int64
	upgrader       websocket.Upgrader
)

type Client struct {
	uid      string
	userId   string
	userType string
	deviceId string
}

type ClientConn struct {
	id       string
	loadTime int64
	ws       *websocket.Conn
	writed   chan bool
	writeSig chan bool
	sending  chan bool
	quiter   chan int
	isClose  bool
	Client
}

type sendingMsg struct {
	key    string
	length int
	msg    []byte
}

func ClientInit() {

	writeWait = 5 * time.Second
	pongWait = 60 * time.Second
	pingPeriod = 50 * time.Second
	maxMessageSize = 512

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     check,
	}
}

func NewClient(w http.ResponseWriter, r *http.Request) {
	cli, err := getClientData(r)
	if err != nil {
		http.Error(w, http.StatusText(403), 403)
		log.Println(err)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	id := cli.deviceId
	c := ClientConn{}
	c.id = id
	c.writed = make(chan bool)
	c.writeSig = make(chan bool, 5)
	c.quiter = make(chan int)
	c.sending = make(chan bool)
	c.ws = ws
	c.Client = *cli
	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))

	c.loadTime = time.Now().Unix()
	log.Println("create client id: ", c.id)
	// add self to controller
	Controller.register <- &c
	go c.run()
}

func check(r *http.Request) bool {
	return true
}

func randStr() string {
	b := make([]byte, 15)
	rand.Read(b)
	str := hex.EncodeToString(b)
	return str
}

func getClientData(r *http.Request) (*Client, error) {
	values := r.URL.Query()
	uid := values.Get("uid")
	userId := values.Get("uid")
	userType := values.Get("userType")
	deviceId := r.Header.Get("d")
	// check query value
	if uid == "" || userType == "" {
		log.Println("request data error")
		err := errors.New("request data error")
		return nil, err
	}
	if deviceId == "" {
		log.Println("request data error device id not found")
		err := errors.New("request data error")
		return nil, err
	}
	if userType != "teacher" && userType != "account" {
		log.Println("request data error")
		err := errors.New("request data error")
		return nil, err
	}
	uid = uid + ":" + userType
	// if user device_id has connect server response 403
	if user, ok := Controller.Users[uid]; ok {
		if cli, ok := user.clients[deviceId]; ok {
			log.Printf("error: user device id has connect uid: %s | deviceId: %s  \n", uid, deviceId)
			err := errors.New("user device id has connect")
			cli.Close()
			return nil, err
		}
	}

	client := Client{
		uid:      uid,
		userId:   userId,
		userType: userType,
		deviceId: deviceId,
	}

	return &client, nil
}

func (this *ClientConn) Write() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("client conn sending closed:", this.id)
		}
	}()
	log.Println("begin to ping client before send msg id:", this.id)
	this.ws.SetWriteDeadline(time.Now().Add(writeWait))
	if err := this.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
		log.Println("ping error id: ", this.id)
		return
	}
	this.sending <- true

	key := Config["UserMsgListPre"].(string) + ":" + this.uid + ":" + this.id
	msgsStr, err := RConn.GetChatMsgByKey(key)
	if err != nil {
		log.Println("error: GetChatMsgByKey key:", key)
		log.Println(err)
		return
	}

	length := len(msgsStr)
	msgs := make([]string, len(msgsStr))[:0]
	for _, msg := range msgsStr {
		if msg != "" {
			msgs = append(msgs, msg)
		}
	}

	if len(msgs) == 0 {
		log.Println("get msg list.length == 0 key:", key)
		// todo rm null chat_id
		return
	}
	jsonStr := "[" + strings.Join(msgs, ", ") + "]"
	jsonStr = "{\"group_chat\":" + jsonStr + "}"

	if this.isClose {
		log.Println("client has closed id: ", this.id)
		return
	}
	msg := []byte(jsonStr)
	log.Printf("begin to write uid: %s | did: %s | msg: %s \n", this.uid, this.Client, msg)

	// sMsg := sendingMsg{}
	// sMsg.msg = msg
	// sMsg.length = length
	// sMsg.key = key
	// this.sending <- &sMsg

	// wite msg by websocket
	this.ws.SetWriteDeadline(time.Now().Add(writeWait))
	if err := this.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Println(err)
		log.Println("write msg fail client will be close id:", this.id)
		// if send msg error close this connection
		this.quiter <- 1
		return
	}
	log.Println("write msg success id:", this.id)
	RConn.RmChatListByKey(key, length)
}

func (this *ClientConn) WriteRepeatLogin() {
	defer func() {
		if err := recover(); err != nil {
			log.Println("client conn sending closed:", this.id)
		}
	//	this.close()
	}()
	log.Println("begin to ping client before send msg id:", this.id)
	this.ws.SetWriteDeadline(time.Now().Add(writeWait))
	if err := this.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
		log.Println("ping error id: ", this.id)
		return
	}
	this.sending <- true
	jsonStr := "{\"repeat_login\":" + "{\"login_time\":1234,\"msg\":\"repeat login other\",\"device_id\":\"1234567\"}" + "}"
	msg := []byte(jsonStr)
	this.ws.SetWriteDeadline(time.Now().Add(writeWait))
	if err := this.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
		log.Println(err)
		log.Println("write repeat login msg fail client will be close id:", this.id)
		// if send msg error close this connection
		this.quiter <- 1
		return
	}
	log.Println("write repeat login msg success id:", this.id)
}

func (this *ClientConn) Close() {
	log.Println("server close client id:", this.id)
	this.isClose = true
	this.ws.SetWriteDeadline(time.Now().Add(writeWait))
	this.ws.WriteMessage(websocket.CloseMessage, []byte{})
	this.quiter <- 0
}

func (this *ClientConn) run() {
	go this.readMsg()
	// go this.sendMsg()
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		this.isClose = true
		ticker.Stop()
		this.ws.Close()
		log.Println("client conn close id:", this.id)
	}()

	this.ws.SetPongHandler(func(string) error {
		this.ws.SetReadDeadline(time.Now().Add(pongWait))
		// if write msg
		select {
		case <-this.sending:
			log.Println("read pong begin to send id:", this.id)
		default:
		}

		return nil
	})

	for {
		select {

		case <-ticker.C:
			// log.Println("ping send msg id:", this.id)
			this.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := this.ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.Println(err)
				log.Println("ping error id: ", this.id)
				return
			}
		case <-this.quiter:
			return
		}
	}
}

// func (this *ClientConn) sendMsg() {
// 	for {
// 		_, ok := <-this.writeSig
// 		if !ok {
// 			return
// 		}
// 		this.Write()
// 		// _, ok = <-this.writed
// 		// if !ok {
// 		// 	return
// 		// }
// 	}
// }

func (this *ClientConn) readMsg() {
	defer func() {
		this.close()
	}()

	for {
		_, msg, err := this.ws.ReadMessage()
		if err != nil {
			log.Println("read msg close id:", this.id)
			log.Println(err)
			break
		}
		log.Println("data from client content:", string(msg))
	}
}

func (this *ClientConn) close() {
	Controller.unregister <- this
	this.isClose = true
	close(this.writed)
	close(this.quiter)
	close(this.sending)
	close(this.writeSig)
	log.Println("client closed id:", this.id)
}
