package main

import (
	"github.com/garyburd/redigo/redis"
)

type RedisConn struct {
	// groupid    chan []byte
	pubSubConn *redis.Conn
	pubsub     *redis.PubSubConn
	connPool   *RedisConnPool
}

type doConn struct {
	conn    *redis.Conn
	usedNum int64
}

type RedisConnPool struct {
	conns []*doConn
	get   chan *doConn
	give  chan *doConn
}

func InitRedisConnPool() *RedisConnPool {
	redisHost := Config["redis"].(string)
	redisDb := Config["redis_db"].(string)
	rcp := RedisConnPool{}
	rcp_num := 10
	rcp.get = make(chan *doConn, rcp_num)
	rcp.give = make(chan *doConn, rcp_num)
	rcp.conns = make([]*doConn, rcp_num)

	for i := 0; i < rcp_num; i++ {
		dc := doConn{}
		dc.usedNum = 0
		conn, err := redis.DialTimeout("tcp", redisHost, 0, 0, 0)
		if err != nil {
			log.Println("error: create redis conn pool err")
			log.Fatal(err)
		}
		if _, err := conn.Do("SELECT", redisDb); err != nil {
			log.Fatal(err)
		}
		dc.conn = &conn
		rcp.conns[i] = &dc
	}

	// set all the conns to channel
	for _, dc := range rcp.conns {
		rcp.get <- dc
	}

	go rcp.rcpRun()
	return &rcp
}

func (this *RedisConnPool) rcpRun() {
	for {
		dc := <-this.give
		dc.usedNum++
		this.get <- dc
	}
}

// create RedisConn
func InitRedisChatGroup() *RedisConn {
	// for pub/sub conn
	pubSubConn, err := redis.DialTimeout("tcp", Config["redis"].(string), 0, 0, 0)
	if err != nil {
		log.Fatal(err)
	}

	pubsub := redis.PubSubConn{pubSubConn}
	pubsub.Subscribe(Config["ChatGroupSubscribePre"].(string))

	this := RedisConn{
		pubSubConn: &pubSubConn,
		pubsub:     &pubsub,
	}
	// this.groupid = make(chan []byte, 100)
	// conn pool
	this.connPool = InitRedisConnPool()
	go this.sub()
	return &this
}

func (this *RedisConn) GetChatMsgByKey(key string) ([]string, error) {
	dcp := <-this.connPool.get
	v, err := (*dcp.conn).Do("sort", key, "get", Config["ChatMsgPre"].(string)+":*")
	this.connPool.give <- dcp
	return redis.Strings(v, err)
}

func (this *RedisConn) GetChatIdsByKey(key string) ([]string, error) {
	dcp := <-this.connPool.get
	vs, err := (*dcp.conn).Do("lrange", key, -1, -1)
	this.connPool.give <- dcp
	return redis.Strings(vs, err)
}

func (this *RedisConn) RmChatListByKey(key string, length int) error {
	dcp := <-this.connPool.get
	_, err := (*dcp.conn).Do("LTRIM", key, length, -1)
	this.connPool.give <- dcp
	return err
}

func (this *RedisConn) GetChatGroupAccounts(groupId string) ([]string, error) {
	dcp := <-this.connPool.get
	k := "ChatGroup:" + groupId + ":account"
	v, err := (*dcp.conn).Do("SMEMBERS", k)
	this.connPool.give <- dcp
	return redis.Strings(v, err)
}

func (this *RedisConn) GetChatGroupTeachers(groupId string) ([]string, error) {
	dcp := <-this.connPool.get
	k := "ChatGroup:" + groupId + ":teacher"
	v, err := (*dcp.conn).Do("SMEMBERS", k)
	this.connPool.give <- dcp
	return redis.Strings(v, err)
}

func (this *RedisConn) sub() {
	defer func() {
		go this.sub()
	}()
	for {
		switch m := this.pubsub.Receive().(type) {
		case redis.Message:
			log.Println("pub groupid: ", string(m.Data))
			// this.groupid <- m.Data
			Controller.groupId <- string(m.Data)

		case redis.Subscription:
			if m.Count == 0 {
				log.Println("done")
				return
			}
		case error:
			log.Println("error : ", m)
			// return
		}
	}
}
