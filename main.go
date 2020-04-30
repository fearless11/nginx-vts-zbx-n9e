package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	c string
	s bool
	u bool
	o string
)

func init() {
	flag.StringVar(&c, "c", "http://10.21.0.11/ngx_status/format/json", "configuration file, default cfg.yaml")
	flag.BoolVar(&s, "s", false, "print serverzone")
	flag.BoolVar(&u, "u", false, "print upstreamzone")
	flag.StringVar(&o, "o", "", "custom parameters")
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

type zbx struct {
	Data []Custom `json:"data"`
}

type Custom struct {
	ServerZone   string `json:"{#SERVERZONE},omitempty"`
	Upstream     string `json:"{#UPSTREAM},omitempty"`
	UpstreamName string `json:"{#UPSTREAMNAME},omitempty"`
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

func discovery() {
	data := run(c)
	z := &zbx{Data: []Custom{}}

	if s {
		for k := range data.ServerZones {
			z.Data = append(z.Data, Custom{ServerZone: strings.Replace(k, "*", "all", -1)})
		}
	}

	if u {
		for k, v := range data.UpstreamZones {
			for _, up := range v {
				z.Data = append(z.Data, Custom{
					UpstreamName: k + "-" + up.Server,
				})
			}
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

func calculation() {
	data1 := run(c)
	time.Sleep(1 * time.Second)
	data2 := run(c)

	if s {
		for k, v := range data1.ServerZones {
			if k == strings.Replace(o, "all", "*", -1) {
				reqs := data2.ServerZones[k].RequestCounter - v.RequestCounter
				fmt.Println(reqs)
			}
		}
	}

	if u {
		for k, upstreams := range data1.UpstreamZones {
			for i, upstream := range upstreams {
				str := strings.Split(o, "-")
				l := len(str)
				if upstream.Server == str[l-1] {
					reqs := data2.UpstreamZones[k][i].RequestCounter - upstream.RequestCounter
					fmt.Println(reqs)
				}
			}
		}
	}
}

func main() {
	flag.Parse()
	if len(o) != 0 {
		calculation()
	} else {
		discovery()
	}
}
