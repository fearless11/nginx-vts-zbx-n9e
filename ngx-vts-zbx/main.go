package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"time"
)

var (
	tmpFile        string
	n9eAPI         string
	option         string
	serverzoneFlag bool
	upstreamFlag   bool
	n9eFlag        bool
	endpoint       string
)

func init() {
	flag.StringVar(&n9eAPI, "c", "http://10.11.100.49/api/transfer/push", "push data to nightingale")
	flag.StringVar(&tmpFile, "f", "/tmp/nginx-request.json", "store nginx-vts data")
	flag.StringVar(&option, "o", "", "zabbix custom parameters")
	flag.BoolVar(&serverzoneFlag, "s", false, "print serverzone")
	flag.BoolVar(&upstreamFlag, "u", false, "print upstream")
	flag.BoolVar(&n9eFlag, "n", false, "push to nightingale")
	flag.StringVar(&endpoint, "p", "10.201.0.11", "assign endpoint")
}

// RPS requests per second
type RPS struct {
	Name    string
	Request int64
	Type    string
}

type zbx struct {
	Data []Custom `json:"data"`
}

type Custom struct {
	ServerZone string `json:"{#SERVERZONE},omitempty"`
	Upstream   string `json:"{#UPSTREAM},omitempty"`
}

type n9e struct {
	Metric      string      `json:"metric"`
	Endpoint    string      `json:"endpoint"`
	Timestamp   int64       `json:"timestamp"`
	Step        int64       `json:"step"`
	Value       interface{} `json:"value"`
	CounterType string      `json:"counterType"`
	Tags        string      `json:"tags"`
}

func readRequestFile() []RPS {
	rps := []RPS{}

	data, err := ioutil.ReadFile(tmpFile)
	if err != nil {
		log.Println(err)
		return rps
	}

	err = json.Unmarshal(data, &rps)
	if err != nil {
		log.Println(err)
		return rps
	}

	return rps
}

func zbxDiscovery() {
	z := &zbx{Data: []Custom{}}
	for _, v := range readRequestFile() {
		if serverzoneFlag && v.Type == "SERVERZONE" {
			z.Data = append(z.Data, Custom{ServerZone: v.Name})
		}

		if upstreamFlag && v.Type == "UPSTREAM" {
			z.Data = append(z.Data, Custom{Upstream: v.Name})
		}
	}

	byt, err := json.Marshal(z)
	if err != nil {
		log.Println(err)
		return
	}

	var out bytes.Buffer
	err = json.Indent(&out, byt, "", "\t")
	if err != nil {
		log.Println(err)
	}
	fmt.Println(out.String())
}

func zbxLLDCalcution() {
	for _, v := range readRequestFile() {
		if v.Name == option {
			fmt.Println(v.Request)
			break
		}
	}
}

func pushNightingale() {

	metrics := make([]*n9e, 0)
	for _, v := range readRequestFile() {

		perfix := "serverzone."
		if v.Type == "UPSTREAM" {
			perfix = "upstream."
		}

		metric := &n9e{}
		metric.Metric = "nginx." + perfix + v.Name
		metric.Endpoint = endpoint
		metric.Value = v.Request
		metric.CounterType = "GAUGE"
		metric.Timestamp = time.Now().Unix()
		metric.Step = 60

		metrics = append(metrics, metric)
	}

	byt, err := json.MarshalIndent(metrics, "", "\t")
	if err != nil {
		log.Println(err)
	}

	fmt.Println(string(byt))

	resp, err := http.Post(n9eAPI, "application/json", bytes.NewReader(byt))
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	respBty, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	reg := regexp.MustCompile("\"dat\":\"ok\"")
	if len(reg.FindAllString(string(respBty), -1)) == 0 {
		log.Println(string(respBty))
	}
}

func main() {
	flag.Parse()
	if len(option) != 0 {
		zbxLLDCalcution()
	} else {
		if serverzoneFlag || upstreamFlag {
			zbxDiscovery()
		}
	}

	if n9eFlag {
		pushNightingale()
	}
}
