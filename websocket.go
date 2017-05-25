package main

import (
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	// "log"
	"net/http"
	"os"
	// "runtime"
)

// global variable

var (
	Config     map[interface{}]interface{}
	Controller *Control
	RConn      *RedisConn
)

// init ConfigMap Controller RConn

func init() {
	// init log first
	logInit()
	// init config
	log.Println("init config ...")
	Config = make(map[interface{}]interface{})
	cfgFile, err := os.Open("ws_chat_push.yml")
	if err != nil {
		log.Println("error: read config file ws_chat_push.yml error")
		log.Fatal(err)
	}
	defer cfgFile.Close()
	buf, err := ioutil.ReadAll(cfgFile)
	if err != nil {
		log.Println("error: read error")
		log.Fatal(err)
	}
	err = yaml.Unmarshal(buf, &Config)
	if err != nil {
		log.Println("error: Unmarshal config file error")
		log.Fatal(err)
	}

	// read flag enviroment
	var env string
	flag.StringVar(&env, "env", "development", "set enviroment")
	flag.Parse()

	// change a enviroment used to config
	if envCfg, ok := Config[env]; ok {
		Config = envCfg.(map[interface{}]interface{})
	} else {
		log.Println("enviroment not found")
		log.Fatal("enviroment not found")
	}
	log.Println("read config complete ...")
	// init config over
	ClientInit()
	RConn = InitRedisChatGroup()
	Controller = InitControl()
}

func main() {
	// runtime.GOMAXPROCS(runtime.NumCPU())
	// log.Println("GOMAXPROCS: ", runtime.NumCPU())

	log.Println("server starting ...")
	http.HandleFunc("/push", func(w http.ResponseWriter, req *http.Request) {
		log.Println("request: ", req.URL.String())
		// clent check ks sig
		if !ksSigCheck(req) {
			// http.Error(w, http.StatusText(403), 403)
			// log.Println("sig error return 403 req: ", req.URL.String())
			// return
		}
		// begin to create a new client
		NewClient(w, req)
	})

	log.Println("server listen: ", Config["port"].(string))
	if err := http.ListenAndServe(":"+Config["port"].(string), nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

	log.Println("end")
}
