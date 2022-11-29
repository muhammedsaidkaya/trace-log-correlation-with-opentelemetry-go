package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"os"
	"strconv"
)

var (
	service     = GetEnv("SERVICE_NAME", "service1")
	environment = GetEnv("ENVIRONMENT", "dev")
	albums      = []album{
		{ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99},
		{ID: "2", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
		{ID: "3", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
		{ID: "4", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
		{ID: "5", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
	}
)

type album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

func init() {
	InitTracer()
}

func main() {
	router := gin.New()
	router.Use(otelgin.Middleware("service1"))
	router.GET("/albums", getAlbums)

	router.Run(":" + GetEnv("APP_PORT", "8080"))
}

func getAlbums(c *gin.Context) {

	fmt.Println(c.Request.Header)

	//span
	ctx := c.Request.Context()
	span := trace.SpanFromContext(ctx)
	defer span.End()
	span.SetAttributes(attribute.Key("trace_id").String(span.SpanContext().TraceID().String()))
	span.SetAttributes(attribute.Key("span_id").String(span.SpanContext().SpanID().String()))
	span.AddEvent("Service1 GetAlbums Function")

	log.WithFields(GetStandardFields(span)).WithContext(ctx).Info(albums)

	c.IndentedJSON(http.StatusOK, albums)
}

func GetStandardFields(span trace.Span) log.Fields {
	return log.Fields{
		"dd.trace_id": span.SpanContext().TraceID().String(),
		"dd.span_id":  span.SpanContext().SpanID().String(),
		"dd.service":  GetEnv("SERVICE_NAME", "service1"),
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
