package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v7"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"

	"github.com/mainnika/habr-adm-quest/task2-backend/lib"
	. "github.com/mainnika/habr-adm-quest/task2-backend/lib/configure"
	. "github.com/mainnika/habr-adm-quest/task2-backend/lib/env"
)

var version = "dev"

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

	pubKey, err := jwtgo.ParseECPublicKeyFromPEM(publicKey)
	if err != nil {
		log.Fatalf("can not parse jwt key: %s", err)
	}

	privKey, err := jwtgo.ParseECPrivateKeyFromPEM(privateKey)
	if err != nil {
		log.Fatalf("can not parse jwt key: %s", err)
	}

	apiserv := &lib.Api{
		Base:       Config.HttpAPI.Base,
		Alg:        alg,
		Pub:        pubKey,
		Priv:       privKey,
		ScoresKey:  Config.Redis.ScoreKey,
		WinnersKey: Config.Redis.WinnersKey,
		Redis:      rediclient,
	}

	httpStart(apiserv)
}
