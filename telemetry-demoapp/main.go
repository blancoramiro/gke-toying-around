package main

import (
	"os"
	"fmt"
	"time"
	"strconv"
	"context"
	"strings"
	"errors"
	"net/http"
	"math/rand"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

type DEMOAPP_CONF struct {
	destination_svcs []string
	destination_svcs_count int
	svc_name string
	otlp_endpoint string
}

var tracer trace.Tracer

var (
        opsProcessed = prometheus.NewCounter(prometheus.CounterOpts{
                Name: "telemetry_demoapp_processed_ops_total",
                Help: "The total number of processed events",
        })
)


func (demoapp_conf *DEMOAPP_CONF) run(w http.ResponseWriter, req *http.Request) {

	var hop int
	var branches int

	ctx := req.Context()
	span := trace.SpanFromContext(ctx)
	propagator := otel.GetTextMapPropagator()
	defer span.End()

	// Hops param
	if _, ok := req.URL.Query()["hop"]; ok {
		if val, err := strconv.Atoi(req.URL.Query()["hop"][0]); err == nil {
			hop = val
		} else {
			log.Error().Msg("Request hop param not int")
			return
		}
		hop++
		if(hop > demoapp_conf.destination_svcs_count) {
			log.Info().
			Str("trace_id", span.SpanContext().TraceID().String()).
			Str("span_id", span.SpanContext().SpanID().String()).
			Int("hop", hop).Int("hops_limit", demoapp_conf.destination_svcs_count).Msg("Max hops reached!")
			fmt.Fprintf(w, "DONE\n")
			return
		}
	} else {
		hop = 1
	}

	// Increase prom metric counter with exemplar for traceID
	rand.Seed(time.Now().UnixNano())
	opsProcessed.(prometheus.ExemplarAdder).AddWithExemplar(rand.Float64()*float64(rand.Intn(5)), prometheus.Labels{"traceID": span.SpanContext().TraceID().String()})

	//Print headers
	//if reqHeadersBytes, err := json.Marshal(req.Header); err != nil {
	//fmt.Println("Could not Marshal Req Headers")
	//} else {
	//fmt.Println(string(reqHeadersBytes))
	//}

	if hop&1 == 0 {
		branches = hop
	} else {
		branches = 1
	}

	log.Info().Int("hop", hop).Int("hops_limit", demoapp_conf.destination_svcs_count).
		Int("branches", branches).
		Str("trace_id", span.SpanContext().TraceID().String()).
		Str("span_id", span.SpanContext().SpanID().String()).
		Msg("New Request")

	// Fake error
	if(rand.Intn(20) < 4) {
		span.SetStatus(codes.Error, "operationThatCouldFail failed")
		span.RecordError(errors.New("Some random error!"))
		log.Error().
		Str("trace_id", span.SpanContext().TraceID().String()).
		Str("span_id", span.SpanContext().SpanID().String()).
		Msg("Random ERROR!")
	}

	//client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	client := http.Client{}

	for branches > 0 && hop <= demoapp_conf.destination_svcs_count {
		ctx, req_span := tracer.Start(ctx, "client_req", trace.WithSpanKind(trace.SpanKindClient)) // <= Making span a client kind
		defer req_span.End()

		// Fake error Client
		if(rand.Intn(20) < 5) {
			req_span.SetStatus(codes.Error, "operationThatCouldFail failed client")
			req_span.RecordError(errors.New("Some random client error!"))
			log.Error().
			Str("trace_id", span.SpanContext().TraceID().String()).
			Str("span_id", req_span.SpanContext().SpanID().String()).
			Msg("Random client ERROR!")
		}

		log.Info().
			Int("branch", branches).
			Str("trace_id", req_span.SpanContext().TraceID().String()).
			Str("span_id", req_span.SpanContext().SpanID().String()).
			Msg("Client Request");
		new_req, _ := http.NewRequestWithContext(ctx, "GET", "http://"+demoapp_conf.destination_svcs[hop]+":8080", nil)

		q := req.URL.Query()
		q.Set("hop", strconv.Itoa(hop))
		new_req.URL.RawQuery = q.Encode()

		propagator.Inject(ctx, propagation.HeaderCarrier(new_req.Header))

		res, err := client.Do(new_req)
		if err != nil {
			log.Error().Msg("Sending request error")
		}
		res.Body.Close()
		hop+=hop
		branches--
	}
	fmt.Fprintf(w, "%v\n", demoapp_conf.svc_name)

}

func health(w http.ResponseWriter, req *http.Request) {

	fmt.Fprintf(w, "OK\n")
}

func initProvider(ctx context.Context, demoapp_conf* DEMOAPP_CONF) (func(context.Context) error, error) {

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(demoapp_conf.svc_name),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of your cluster. If you run the app inside k8s, then you can
	// probably connect directly to the service through dns
	//ctx, cancel := context.WithTimeout(ctx, time.Second)
	//defer cancel()
	conn, err := grpc.DialContext(ctx, demoapp_conf.otlp_endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator B3
	p := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader | b3.B3SingleHeader))
	otel.SetTextMapPropagator(p)

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}


func main() {

	ctx := context.Background()

	demoapp_conf := &DEMOAPP_CONF{}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if val, ok := os.LookupEnv("DEMOAPP_OTLP_ENDPOINT"); ok {
		demoapp_conf.otlp_endpoint = val
	} else {
		log.Error().Msg("Missing DEMOAPP_OTLP_ENDPOINT")
		return
	}

	if val, ok := os.LookupEnv("DEMOAPP_DESTINATION_SVCS"); ok {
		demoapp_conf.destination_svcs = strings.Split(val, ",")
		demoapp_conf.destination_svcs_count = len(demoapp_conf.destination_svcs) - 1
	} else {
		log.Error().Msg("Missing DEMOAPP_DESTINATION_SVCS")
		return
	}

	if demoapp_conf.destination_svcs_count < 1 {
		log.Error().Msg("Need at least 2 svcs in DEMOAPP_DESTINATION_SVCS")
		return
	}

	if val, ok := os.LookupEnv("DEMOAPP_SERVICENAME"); ok {
		demoapp_conf.svc_name = val
	} else {
		log.Error().Msg("Missing DEMOAPP_SERVICENAME")
		return
	}

	shutdown, err := initProvider(ctx, demoapp_conf)
	if err != nil {
		log.Error().Err(err).Msg("")
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("")
		}
	}()

	tracer = otel.Tracer("telemetry-tracer")

	prometheus.MustRegister(opsProcessed)

	log.Info().
		Int("MAX HOPS", demoapp_conf.destination_svcs_count).
		Msg("Running...")

		otelHandler := otelhttp.NewHandler(http.HandlerFunc(demoapp_conf.run), demoapp_conf.svc_name[8:])

	http.Handle("/", otelHandler)
	http.HandleFunc("/health", health)
	http.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{ 
			// Opt into OpenMetrics to support exemplars.
			EnableOpenMetrics: true }))

	http.ListenAndServe(":8080", nil)
}

//Ver 1.0.2
