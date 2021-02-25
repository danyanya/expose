package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	envconfig "github.com/kelseyhightower/envconfig"
	"github.com/wimark/mongo"
)

var eConfig env
var db *mongo.DB

type env struct {
	MongoAddr string `envconfig:"MONGO_ADDR" default:"db"`
	ServeAddr string `envconfig:"SERVE_ADDR" default:":8081"`
}

func init() {
	log.Println("init")

	err := envconfig.Process("", &eConfig)
	if err != nil {
		panic(err.Error())
	}

	db, err = mongo.NewConnectionWithTimeout(eConfig.MongoAddr, 2*time.Minute)
	if err != nil {
		panic(err.Error())
	}
}

func main() {
	log.Println("start")

	http.Handle("/", &proxyHandler{})
	err := http.ListenAndServe(eConfig.ServeAddr, nil)
	if err != nil {
		panic(err)
	}
}

type proxyHandler struct {
	p map[string]*httputil.ReverseProxy
}

func (ph *proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	a := strings.Split(host, ".")
	mac := a[0]
	fmt.Println(r.URL, mac)

	host, err := findHost(db, mac) // be care with concurency
	if host == "" {
		host = "empty"
	}
	if err != nil {
		fmt.Println(r.URL, mac, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("502: Error with " + host + " host"))
		return
	}

	fmt.Println(r.URL, mac, host)

	// if nil - create new proxy
	if ph.p[mac] == nil {
		if len(ph.p) == 0 {
			ph.p = map[string]*httputil.ReverseProxy{}
		}
		ph.p[mac] = httputil.NewSingleHostReverseProxy(&url.URL{
			Scheme: "https",
			Host:   host,
		})
		ph.p[mac].Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	// either serve created before
	ph.p[mac].ServeHTTP(w, r)
}

const coll = "cpes"

type cpeHost struct {
	ID string `json:"id" bson:"_id"`

	State struct {
		L2TPConfig struct {
			Addr string `json:"ip" bson:"local_addr"`
		} `json:"l2tp" bson:"l2tp_state"`
	} `json:"state" bson:"state"`
}

func findHost(db *mongo.DB, mac string) (string, error) {
	q := mongo.M{
		"_id": mongo.M{
			"$regex": mac,
		},
		// "connected": true,
		"state.l2tp_state.tunnel_type": mongo.M{
			"$regex": "ipsec",
		},
	}
	var cpe = cpeHost{}
	fmt.Println(q)
	var err = db.FindWithQueryOne(coll, q, &cpe)
	return cpe.State.L2TPConfig.Addr, err
}
