package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/docker/docker/client"
	"github.com/go-redis/redis/v7"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"

	"github.com/mainnika/habr-adm-quest/task3-backend/lib"
	. "github.com/mainnika/habr-adm-quest/task3-backend/lib/configure"
	. "github.com/mainnika/habr-adm-quest/task3-backend/lib/env"
)

var version = "dev"
var exit = make(chan int)

var (
	getVersion bool
)

func init() {

	if IsDevelopment {
		log.SetLevel(log.DebugLevel)
		log.Debug("debug mode")
	}
}

func httpStart(apiserv *lib.Api) (httpserv *fasthttp.Server) {

	httpserv = &fasthttp.Server{
		Logger:           log.StandardLogger(),
		Handler:          apiserv.GetHandler(),
		DisableKeepalive: true,
	}

	lis, err := net.Listen("tcp", Config.HttpAPI.Addr)
	if err != nil {
		log.Fatalf("http listen error: %v", err)
	}

	err = httpserv.Serve(lis)
	if err != nil {
		log.Fatalf("http serve error: %v", err)
	}

	return
}

func taskStart(taskserv *lib.Server) {

	lis, err := net.Listen("tcp", Config.Task.Addr)
	if err != nil {
		log.Fatalf("task listen error: %v", err)
	}

	err = taskserv.Serve(lis)
	if err != nil {
		log.Fatalf("task serve error: %v", err)
	}

	return
}

func main() {

	flag.BoolVar(&getVersion, "v", false, "version")
	flag.Parse()

	if getVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	log.Debugf("version: %s", version)
	log.Debugf("cfg: %v", Config)

	rediclient := redis.NewClient(&redis.Options{
		Addr:     Config.Redis.Addr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	docklient, err := client.NewEnvClient()
	if err != nil {
		log.Fatalf("can not init docklient: %s", err)
	}

	pubKey, err := jwtgo.ParseECPublicKeyFromPEM(publicKey)
	if err != nil {
		log.Fatalf("can not parse jwt key: %s", err)
	}

	privKey, err := jwtgo.ParseECPrivateKeyFromPEM(privateKey)
	if err != nil {
		log.Fatalf("can not parse jwt key: %s", err)
	}

	apiserv := &lib.Api{
		Base:   Config.HttpAPI.Base,
		Docker: docklient,
		Redis:  rediclient,
	}

	taskserv := &lib.Server{
		Alg:          alg,
		Pub:          pubKey,
		Priv:         privKey,
		Docker:       docklient,
		WinnersKey:   Config.Redis.WinnersKey,
		Redis:        rediclient,
		ClientsLimit: uint32(Config.Task.Clients),
	}

	go httpStart(apiserv)
	go taskStart(taskserv)

	code := <-exit
	if code > 0 {
		os.Exit(code)
	}
}
