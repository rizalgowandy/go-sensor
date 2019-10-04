package main

import (
    "fmt"
    "time"
    "net/http"
    "github.com/gin-gonic/gin"
    instana "github.com/instana/go-sensor"
)

var sensor = instana.NewSensor("Gin")

func health(ctx *gin.Context) {
    ctx.Writer.WriteString("OK")
}

func api(ctx *gin.Context) {
    name := ctx.Param("bar")
    time.Sleep(300 * time.Millisecond)
    ctx.String(http.StatusOK, "foo = %s", name)
}

func badRequest(ctx *gin.Context) {
    time.Sleep(250 * time.Millisecond)
    ctx.String(http.StatusBadRequest, "bad request")
}

func bang(ctx *gin.Context) {
    time.Sleep(250 * time.Millisecond)
    ctx.String(http.StatusInternalServerError, "BANG")
}

func main() {
    fmt.Println("Starting")

    router := gin.Default()
    // Instana middleware MUST be the first
    router.Use(sensor.GinWrap())

    router.GET("/health", health)
    router.GET("/foo/:bar", api)
    router.GET("/bad", badRequest)
    router.GET("/bang", bang)

    // fire it up
    router.Run(":8080")
}

