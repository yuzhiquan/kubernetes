/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"k8s.io/component-base/version"
	"k8s.io/kubernetes/pkg/probe"
)

var (
	errGrpcNotServing = errors.New("GRPC_STATUS_NOT_SERVING")
)

// Prober is an interface that defines the Probe function for doing GRPC readiness/liveness checks.
type Prober interface {
	Probe(host, service string, port int, timeout time.Duration, opts ...grpc.DialOption) (probe.Result, string, error)
}

type grpcProber struct {
}

// New Prober for execute grpc probe
func New() Prober {
	return grpcProber{}
}

// Probe executes a grpc call to check the liveness/readiness of container.
// Returns the Result status, command output, and errors if any.
func (p grpcProber) Probe(host, service string, port int, timeout time.Duration, opts ...grpc.DialOption) (probe.Result, string, error) {
	v := version.Get()

	md := metadata.New(map[string]string{
		"User-Agent": fmt.Sprintf("kubernetes/%s.%s", v.Major, v.Minor),
	})

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	defer cancel()

	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", host, port), opts...)

	if err != nil {
		return probe.Failure, fmt.Sprintf("grpc probe DialContext() failure: %s", err.Error()), err
	}

	defer func() {
		_ = conn.Close()
	}()

	client := grpchealth.NewHealthClient(conn)

	resp, err := client.Check(metadata.NewOutgoingContext(ctx, md), &grpchealth.HealthCheckRequest{
		Service: service,
	})

	if err != nil {
		return probe.Failure, fmt.Sprintf("GRPC probe failed make watchclient with error: %s", err.Error()), err
	}

	if resp.Status != grpchealth.HealthCheckResponse_SERVING {
		return probe.Failure, fmt.Sprintf("GRPC probe failed with status: %s", resp.Status.String()), errGrpcNotServing
	}

	return probe.Success, fmt.Sprintf("GRPC probe success"), nil
}
