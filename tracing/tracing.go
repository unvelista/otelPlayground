/* -------------------------------------------------------------------

Copyright (C) CampusGenius GmbH 2021 * ALL RIGHTS RESERVED

 This software is protected by the inclusion of the above copyright
 notice. This software may not be provided or otherwise made available
 to, or used by, any other person. No title to or ownership of the
 software is  hereby  transferred.
 The information contained in this document is considered the
 CONFIDENTIAL and PROPRIETARY information of CampusGenius GmbH and
 may not be disclosed or discussed with anyone who is not employed
 by CampusGenius GmbH, unless the individual / company
 (i) has an express need to know such information, and
 (ii) disclosure of information is subject to the terms of a duly
 executed Source-Code-License or Confidentiality and Non-Disclosure
 Agreement between CampusGenius GmbH and the individual / company.

   ---------------------------------------------------------------- */

package tracing

import (
	"context"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
)

func InitProvider(serviceName string) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	otelAgentAddr, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !ok {
		otelAgentAddr = "0.0.0.0:4317"
	}
	log.Println("Running collector endpoint on", otelAgentAddr)

	traceClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otelAgentAddr),
		otlptracegrpc.WithDialOption(grpc.WithBlock()))
	traceExp, err := otlptrace.New(ctx, traceClient)
	if err != nil {
		log.Printf("ERROR: generating trace exporter failed:%+v\n", err)
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		log.Printf("ERROR: generating resource failed:%+v\n", err)
		return nil, err
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExp)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(tracerProvider)

	return tracerProvider, err
}

func Shutdown(tp *trace.TracerProvider) {
	log.Println("Shuting down tracing...")
	err := tp.Shutdown(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
