package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	url     string
	tmpFile string
)

func init() {
	flag.StringVar(&url, "u", "http://10.21.0.11/ngx_status/format/json", "the url of nginx-vts-json")
	flag.StringVar(&tmpFile, "f", "/tmp/nginx-request.json", "store nginx-vts data")
}

type Response struct {
	ServerZones   map[string]ServerName     `json:"serverzones,omitempty"`
	UpstreamZones map[string][]UpstreamName `json:"upstreamZones,omitempty"`
}

type ServerName struct {
	RequestCounter int64 `json:"requestCounter"`
}

type UpstreamName struct {
	Server         string `json:"server"`
	RequestCounter int64  `json:"requestCounter"`
}

// RPS requests per second
type RPS struct {
	Name    string
	Request int64
	Type    string
}

func run(url string) *Response {
	data := &Response{}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(3) * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		log.Println(err)
		return data
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return data
	}

	err = json.Unmarshal([]byte(body), data)
	if err != nil {
		log.Println(err)
		return data
	}

	return data
}

func caclutionRequest() {
	data1 := run(url)
	time.Sleep(1 * time.Second)
	data2 := run(url)

	rps := []RPS{}

	file, err := os.OpenFile(tmpFile, os.O_TRUNC|os.O_CREATE|os.O_RDWR, 0664)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	for k, v := range data1.ServerZones {
		reqs := data2.ServerZones[k].RequestCounter - v.RequestCounter
		rps = append(rps, RPS{
			Name:    strings.Replace(k, "*", "all", -1),
			Request: reqs,
			Type:    "SERVERZONE",
		})
	}

	for k, upstreams := range data1.UpstreamZones {
		for i, upstream := range upstreams {
			reqs := data2.UpstreamZones[k][i].RequestCounter - upstream.RequestCounter
			rps = append(rps, RPS{
				Name:    k + "-" + upstream.Server,
				Request: reqs,
				Type:    "UPSTREAM",
			})
		}
	}

	byt, err := json.Marshal(rps)
	if err != nil {
		log.Println(err)
		return
	}

	_, err = file.Write(byt)
	if err != nil {
		log.Println(err)
		return
	}
}

func main() {
	flag.Parse()
	caclutionRequest()
	// t := time.NewTicker(1 * time.Minute)
	// defer t.Stop()
	// for {
	// 	caclutionRequest()
	// 	<-t.C
	// }
}
