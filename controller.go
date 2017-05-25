package main

import (
	// "fmt"
	// "log"
	// "encoding/json"
	// "errors"
	// "strconv"
	// "strings"
	"time"
)

type Control struct {
	Clients    map[string]*ClientConn
	Users      map[string]*User
	register   chan *ClientConn
	unregister chan *ClientConn
	userId     chan string
	user       chan *User
	groupId    chan string
}

type User struct {
	uid        string
	userType   string
	pushSignal chan bool
	clients    map[string]*ClientConn
}

type PushData struct {
	ChatIds  []int  `json:"chat_ids"`
	UserId   int    `json:"user_id"`
	UserType string `json:"user_type"`
}

func InitControl() *Control {
	this := Control{}
	this.Clients = make(map[string]*ClientConn)
	this.Users = map[string]*User{}
	this.register = make(chan *ClientConn)
	this.unregister = make(chan *ClientConn)
	this.groupId = make(chan string, 100000)
	this.user = make(chan *User)
	this.userId = make(chan string)
	// this.groupSignal = make(map[string][]bool)
	go this.run()
	go this.sub()
	return &this
}

func (this *Control) run() {
	// once only register or unregister to keep data uniform
	for {
		select {
		case c, ok := <-this.unregister:
			if !ok {
				log.Println("server error contorller unregister closed")
				log.Panic("server error contorller unregister closed")
			}
			if c == nil {
				log.Println(" a nil *ClineConn error")
				continue
			}
			delete(this.Clients, c.id)
			if this.Users[c.uid] != nil {
				delete(this.Users[c.uid].clients, c.id)
			}
			user := this.Users[c.uid]
			if user != nil && len(user.clients) == 0 {
				delete(this.Users, c.uid)
			}

			now := time.Now().Unix()
			onlineTime := now - c.loadTime
			log.Printf("client offline uid: %s | userType: %s | onlineTime: %d s | id: %s",
				c.uid, c.userType, onlineTime, c.id)
			go onlineNotice(&c.Client, 0)

		case c, ok := <-this.register:
			if !ok {
				log.Println("server error contorller register closed")
				log.Panic("server error contorller register closed")
			}
			if c == nil {
				log.Println(" a nil *ClineConn error")
				continue
			}
			
			this.Clients[c.id] = c
			/*
			if user, ok := this.Users[c.uid]; ok {
				user.clients[c.id] = c
			} else {
				this.Users[c.uid] = NewUser(c)
			}*/
			if user, ok := this.Users[c.uid]; c.userType == "account" && ok {
				log.Println("TOMX1",c.id,len(user.clients))
				user.clients[c.id] = c
				log.Println("TOMX2",len(user.clients))
				for key, cli := range user.clients {
					log.Println("TOMX3",key,len(user.clients))
					if key != c.id{
						log.Println("TOMX4",key,len(user.clients))
						go cli.WriteRepeatLogin()
					}
				}
				log.Println("TOMX5")
				//go user.writeRepeatLoginMsg(c.id)
			}else {
				this.Users[c.uid] = NewUser(c)
			}

			user := this.Users[c.uid]
			log.Printf("client online uid: %s | userType: %s | id: %s",
				c.uid, c.userType, c.id)

			go user.writeMsg()
			go onlineNotice(&c.Client, 1)

		// get user by user_id
		case uid := <-this.userId:
			if user, ok := this.Users[uid]; ok {
				this.user <- user
			} else {
				this.user <- nil
			}
		}
	}
}

func (this *Control) getUser(uid string) *User {
	this.userId <- uid
	user := <-this.user
	return user
}

// group broadcast
func (this *Control) sub() {
	// groupSignal := map[string]int{}
	// gidChan := make(chan string, 1000000)
	// go func() {
	// 	for {
	// 		gid := <-this.groupId
	// 		if bo, ok := groupSignal[gid]; ok {
	// 			if bo == 0 {
	// 				groupSignal[gid] = 1
	// 				// gidChan <- gid
	// 			}
	// 		} else {
	// 			groupSignal[gid] = true
	// 			gidChan <- gid
	// 		}
	// 	}
	// }()

	// go func() {
	for {
		gid := <-this.groupId
		this.GroupBroadcast(gid)
	}

	// }()

}

func (this *Control) GroupBroadcast(gid string) {
	accountIds, err := RConn.GetChatGroupAccounts(gid)
	if err != nil {
		log.Println(err)
		return
	}
	teacherIds, err := RConn.GetChatGroupTeachers(gid)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("group_id: %s | teacher_ids: %s | account_ids: %s", gid, teacherIds, accountIds)
	for _, accountId := range accountIds {
		k := accountId + ":account"
		// user, ok := Controller.Users[k]
		user := Controller.getUser(k)
		if user == nil {
			continue
		}
		// go func(user *User) {
		user.writeMsg()
		// }(user)
	}

	for _, teacherId := range teacherIds {
		k := teacherId + ":teacher"
		user := Controller.getUser(k)
		if user == nil {
			continue
		}
		// go func(user *User) {
		user.writeMsg()
		// }(user)
	}
}

func NewUser(c *ClientConn) *User {
	user := User{}
	user.clients = map[string]*ClientConn{c.id: c}
	user.uid = c.uid
	user.userType = c.userType
	return &user
}

func (this *User) writeMsg() {
	for _, cli := range this.clients {
		go func(cli *ClientConn) {
			defer func() {
				if err := recover(); err != nil && cli != nil {
					log.Println("client conn writeSig closed:", cli.id)
				}
			}()

			cli.Write()
		}(cli)
	}
}
