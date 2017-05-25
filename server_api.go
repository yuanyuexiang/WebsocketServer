package main

import (
	"bytes"
	// "log"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type ApiResponse struct {
	Code string `json:"code"`
}

type ApiLimit struct {
	limit chan bool
}

var apiLimit *ApiLimit

func init() {
	apiLimit = &ApiLimit{}
	length := 2
	apiLimit.limit = make(chan bool, length)
	for i := 0; i < length; i++ {
		apiLimit.limit <- true
	}
}

func (this *ApiLimit) apply() {
	<-this.limit
}

func (this *ApiLimit) over() {
	this.limit <- true
}

func ksSigCheck(req *http.Request) bool {
	// block this func if limit is empty
	apiLimit.apply()
	defer apiLimit.over()
	// return true

	sig := req.Header.Get("KS-Sig")
	reqPath := req.URL.Path
	query := req.URL.RawQuery
	apiVersion := req.Header.Get("API-Version")

	host := "http://" + Config["pushsServer"].(string)
	path := "/ws/signature"

	urlStr := host + path
	log.Println(urlStr)
	log.Println("post check sig api data: ", sig)
	log.Println("post check path data: ", reqPath)
	log.Println("post check query data: ", query)
	log.Println("post check query version: ", apiVersion)

	res, err := http.PostForm(urlStr, url.Values{
		"path":        {reqPath},
		"signature":   {sig},
		"query":       {query},
		"api_version": {apiVersion},
	})

	if err != nil {
		log.Println(err)
		return false
	}
	if res.StatusCode == 500 {
		log.Println("api 500")
		return false
	}

	log.Println(res.StatusCode)
	body, _ := ioutil.ReadAll(res.Body)
	bodystr := string(body)
	log.Println(bodystr)
	resjson := ApiResponse{}
	if err := json.Unmarshal(body, &resjson); err != nil {
		log.Println(err)
		return false
	}
	if resjson.Code == "e0000" {
		return true
	} else {
		return false
	}
}

func onlineNotice(cli *Client, isOnline int) {
	// block this func if limit is empty
	apiLimit.apply()
	defer apiLimit.over()
	// return

	host := "http://" + Config["pushsServer"].(string)
	path := "/ws/user/status"
	urlStr := host + path
	log.Println(urlStr)
	log.Printf("post online notice api user_id: %s | device_id: %s | user_type: %s | online: %d \n",
		cli.userId, cli.deviceId, cli.userType, isOnline)

	res, err := http.PostForm(urlStr, url.Values{
		"user_id":   {cli.userId},
		"device_id": {cli.deviceId},
		"user_type": {cli.userType},
		"online":    {strconv.Itoa(isOnline)},
	})
	if err != nil {
		log.Println(err)
		return
	}
	if res.StatusCode == 500 {
		log.Println("api 500")
		return
	}
	log.Println(res.StatusCode)
	body, _ := ioutil.ReadAll(res.Body)
	bodystr := string(body)
	log.Println(bodystr)
	resjson := ApiResponse{}
	if err := json.Unmarshal(body, &resjson); err != nil {
		log.Println(err)
		return
	}
}

func pushChatMsgList(d []byte) {
	// block this func if limit is empty
	apiLimit.apply()
	defer apiLimit.over()

	host := "http://" + Config["pushsServer"].(string)
	path := "/ws/chatbox/offline_push"

	urlStr := host + path
	log.Println(urlStr)
	log.Println("post push api data: ", string(d))
	buf := bytes.NewBuffer(d)
	client := http.Client{}
	req, _ := http.NewRequest("POST", urlStr, buf)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(res.StatusCode)
	body, _ := ioutil.ReadAll(res.Body)
	bodystr := string(body)
	log.Println(bodystr)
}
