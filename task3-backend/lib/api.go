package lib

import (
	"context"
	"net/http"
	"strings"

	"github.com/docker/docker/client"
	"github.com/go-redis/redis/v7"
	routing "github.com/jackwhelpton/fasthttp-routing/v2"
	"github.com/jackwhelpton/fasthttp-routing/v2/access"
	"github.com/jackwhelpton/fasthttp-routing/v2/cors"
	"github.com/jackwhelpton/fasthttp-routing/v2/fault"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
)

const (
	URLHealthz = "/healthz"
)

type Api struct {
	Base   string
	Docker *client.Client
	Redis  *redis.Client
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

	api.Get(URLHealthz, a.healthCheck)

	return router.HandleRequest
}

func (a *Api) healthCheck(ctx *routing.Context) (err error) {

	_, err = a.Redis.Ping().Result()
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		return
	}

	_, err = a.Docker.Ping(context.Background())
	if err != nil {
		ctx.SetStatusCode(http.StatusInternalServerError)
		return
	}

	ctx.Response.Header.Set("Content-Type", "application/json")
	_, err = ctx.WriteString("{\"health\":\"ok\"}")

	return
}

