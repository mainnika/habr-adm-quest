package lib

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v7"
	routing "github.com/jackwhelpton/fasthttp-routing/v2"
	"github.com/jackwhelpton/fasthttp-routing/v2/access"
	"github.com/jackwhelpton/fasthttp-routing/v2/cors"
	"github.com/jackwhelpton/fasthttp-routing/v2/fault"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

const (
	URL        = "/check"
	URLHealthz = "/healthz"
)

type Api struct {
	Base string

	Alg  string
	Pub  *ecdsa.PublicKey
	Priv *ecdsa.PrivateKey

	ScoresKey  string
	WinnersKey string
	Redis      *redis.Client
}

func (a *Api) GetHandler() fasthttp.RequestHandler {

	crs := cors.Options{
		AllowOrigins:     "*",
		AllowHeaders:     "*",
		AllowMethods:     "*",
		AllowCredentials: true,
	}

	router := routing.New()
	router.Use(
		access.Logger(log.Debugf),
		cors.Handler(crs),
		fault.PanicHandler(log.Warnf),
	)

	base := strings.TrimSuffix(a.Base, "/")
	api := router.Group(base)

	api.Post(URL, a.checkScore)
	api.Get(URLHealthz, a.healthCheck)

	return router.HandleRequest
}

func (a *Api) healthCheck(ctx *routing.Context) (err error) {

	_, err = a.Redis.Ping().Result()
	if err != nil {
		return routing.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	ctx.Response.Header.Set("Content-Type", "application/json")
	_, err = ctx.WriteString("{\"health\":\"ok\"}")
	if err != nil {
		return routing.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return
}

func (a *Api) checkScore(ctx *routing.Context) (err error) {

	type Score struct {
		Key   string `json:"key"`
		Score int    `json:"score"`
	}

	type Response struct {
		Top []redis.Z `json:"top"`
		Msg []string  `json:"msg"`
	}

	now := time.Now()
	body := ctx.Request.Body()
	claims := &jwtgo.StandardClaims{}
	score := &Score{}

	err = json.Unmarshal(body, score)
	if err != nil {
		return routing.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	_, err = jwtgo.ParseWithClaims(score.Key, claims, a.getKey)
	if err != nil {
		return routing.NewHTTPError(http.StatusForbidden, err.Error())
	}

	nowunix := fmt.Sprintf("%d", now.UnixNano()) // 1578057345996508551
	member := fmt.Sprintf("%s:%s", nowunix, claims.Subject)

	if len(nowunix) != 19 {
		panic(errors.New("o rly"))
	}

	_, err = a.Redis.ZAddNX(a.ScoresKey, &redis.Z{Score: float64(score.Score), Member: member}).Result()
	if err != nil {
		return routing.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	zz, err := a.Redis.ZRevRangeWithScores(a.ScoresKey, 0, 2).Result()
	if err != nil {
		return routing.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	zz = append(zz, redis.Z{
		Score:  201920192019,
		Member: fmt.Sprintf("%d:%s", time.Unix(claims.IssuedAt, now.Unix()).UnixNano(), claims.Issuer),
	})

	sort.Slice(zz, func(i, j int) bool {
		return zz[i].Score > zz[j].Score
	})

	resp := &Response{Top: zz}

	if int(zz[0].Score) > score.Score {
		resp.Msg = []string{
			`ух, хорошая игра, но нужно стараться еще!`,
		}
	} else {
		resp.Msg = []string{
			`поздравляю с победой`,
			`следующий уровень ищи здесь:`,
			`nc 1451e73305328869824eda4c81a75cb5.mehrweg.ga 31337`,
			`отправь туда ключ чтобы открыть`,
		}

		_, err = a.Redis.SAdd(a.WinnersKey, claims.Id).Result()
		if err != nil {
			return routing.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	body, err = json.Marshal(resp)
	if err != nil {
		return routing.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	ctx.Response.Header.Set("Content-Type", "application/json")
	err = ctx.Write(body)
	if err != nil {
		return routing.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return
}

func (a *Api) getPrivKey(t *jwtgo.Token) (interface{}, error) {
	if _, ok := t.Method.(*jwtgo.SigningMethodECDSA); !ok {
		return nil, errors.New("unexpected signing method")
	}
	return a.Priv, nil
}

func (a *Api) getKey(t *jwtgo.Token) (interface{}, error) {
	if _, ok := t.Method.(*jwtgo.SigningMethodECDSA); !ok {
		return nil, errors.New("unexpected signing method")
	}
	return a.Pub, nil
}
