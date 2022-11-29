package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

var (
	service     = GetEnv("SERVICE_NAME", "service2")
	environment = GetEnv("ENVIRONMENT", "dev")
)

type album struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

func init() {
	InitTracer()
}

func main() {
	router := gin.New()
	router.Use(otelgin.Middleware("service2"))
	router.GET("/albums/:id", getAlbumById)

	router.Run(":" + GetEnv("APP_PORT", "8080"))
}

func getAlbumById(c *gin.Context) {
	id := c.Param("id")

	//span
	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	defer span.End()
	span.SetAttributes(attribute.Key("id").String(id))
	span.SetAttributes(attribute.Key("trace_id").String(span.SpanContext().TraceID().String()))
	span.SetAttributes(attribute.Key("span_id").String(span.SpanContext().SpanID().String()))
	span.AddEvent("Service2 GetById Function")

	//http client - request to service1
	albums := getAlbums(ctx)
	if obj, err := filterAlbumsById(albums, id); err != nil {
		span.SetStatus(codes.Error, "Error")
		span.SetAttributes(attribute.Key("error").String(err.Error()))
		log.WithFields(GetStandardFields(span)).WithContext(ctx).Error(err.Error())
		c.IndentedJSON(http.StatusNotFound, gin.H{"data": obj})
	} else {
		log.WithFields(GetStandardFields(span)).WithContext(ctx).Info(obj)
		c.IndentedJSON(http.StatusOK, gin.H{"data": obj})
	}
}

func filterAlbumsById(albums []album, id string) (interface{}, error) {
	for _, obj := range albums {
		if obj.ID == id {
			return obj, nil
		}
	}
	return nil, errors.New("not found")
}

func getAlbums(ctx context.Context) []album {
	requestURL := fmt.Sprintf("http://service1:8080/albums")
	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, _ := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	res, err := client.Do(req)
	if err != nil {
		fmt.Printf("error making http request: %s\n", err)
		os.Exit(1)
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("client: could not read response body: %s\n", err)
		os.Exit(1)
	}
	var albums []album
	json.Unmarshal(body, &albums)
	return albums
}

func GetStandardFields(span trace.Span) log.Fields {
	return log.Fields{
		"dd.trace_id": span.SpanContext().TraceID().String(),
		"dd.span_id":  span.SpanContext().SpanID().String(),
		"dd.service":  GetEnv("SERVICE_NAME", "service2"),
		"dd.env":      GetEnv("ENVIRONMENT", "dev"),
	}
}

func ConvertTraceID(id string) string {
	if len(id) < 16 {
		return ""
	}
	if len(id) > 16 {
		id = id[16:]
	}
	intValue, err := strconv.ParseUint(id, 16, 64)
	if err != nil {
		return ""
	}
	return strconv.FormatUint(intValue, 10)
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func InitTracer() {
	tp, err := tracerProvider("http://" + GetEnv("JAEGER_URL", "localhost:14268") + "/api/traces")
	if err != nil {
		log.Fatal(err)
	}
	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

func tracerProvider(url string) (*tracesdk.TracerProvider, error) {
	// Create the Jaeger exporter
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(url)))
	if err != nil {
		return nil, err
	}
	tp := tracesdk.NewTracerProvider(
		// Always be sure to batch in production.
		tracesdk.WithBatcher(exp),
		// Record information about this application in a Resource.
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(service),
			attribute.String("environment", environment),
		)),
	)
	return tp, nil
}
