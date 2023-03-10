// Copyright 2018, 2019 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/cls"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/spanhelper"

	"github.com/networkservicemesh/networkservicemesh/pkg/tools/jaeger"

	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/pkg/tools"
	"github.com/networkservicemesh/networkservicemesh/sdk/common"
)

const (
	// ConnectTimeout - a default connection timeout
	ConnectTimeout = time.Minute
	// ConnectionRetry - A number of retries for establish a network service, default == 10
	ConnectionRetry = 18
	// RequestDelay - A delay between attempts, default = 5sec
	RequestDelay = time.Second * 10
)

// NsmClient is the NSM client struct
type NsmClient struct {
	*common.NsmConnection
	ClientNetworkService string
	ClientLabels         map[string]string
	OutgoingConnections  []*connection.Connection
	NscInterfaceName     string
	tracerCloser         io.Closer
}

// Connect with no retry and delay
func (nsmc *NsmClient) Connect(ctx context.Context, name, mechanism, description string) (*connection.Connection, error) {
	return nsmc.ConnectRetry(ctx, name, mechanism, description, 1, 0)
}

// Connect implements the business logic
func (nsmc *NsmClient) ConnectRetry(ctx context.Context, name, mechanism, description string, retryCount int, retryDelay time.Duration) (*connection.Connection, error) {
	span := spanhelper.FromContext(ctx, "nsmClient.Connect")
	defer span.Finish()

	span.Logger().Infof("Initiating an outgoing connection.")
	nsmc.Lock()
	defer nsmc.Unlock()

	if nsmc.NscInterfaceName != "" {
		// The environment variable will override local call parameters
		name = nsmc.NscInterfaceName
	}

	outgoingMechanism, err := common.NewMechanism(cls.LOCAL, mechanism, name, description)

	span.LogObject("Selected mechanism", outgoingMechanism)

	if err != nil {
		err = errors.Wrap(err, "failure to prepare the outgoing mechanism preference with error")
		span.LogError(err)
		return nil, err
	}

	routes := []*connectioncontext.Route{}
	for _, r := range nsmc.Configuration.Routes {
		routes = append(routes, &connectioncontext.Route{
			Prefix: r,
		})
	}

	outgoingRequest := &networkservice.NetworkServiceRequest{
		Connection: &connection.Connection{
			NetworkService: nsmc.Configuration.ClientNetworkService,
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					SrcIpRequired: true,
					DstIpRequired: true,
					SrcRoutes:     routes,
				},
			},
			Labels: nsmc.ClientLabels,
		},
		MechanismPreferences: []*connection.Mechanism{
			outgoingMechanism,
		},
	}
	var outgoingConnection *connection.Connection
	maxRetry := retryCount
	for retryCount >= 0 {
		var attemptSpan = spanhelper.FromContext(span.Context(), fmt.Sprintf("nsmClient.Connect.attempt:%v", maxRetry-retryCount))
		defer attemptSpan.Finish()

		attempCtx, cancelProc := context.WithTimeout(attemptSpan.Context(), ConnectTimeout)
		defer cancelProc()

		attemptLogger := attemptSpan.Logger()
		attemptLogger.Infof("Requesting %v", outgoingRequest)
		outgoingConnection, err = nsmc.NsClient.Request(attempCtx, outgoingRequest)

		if err != nil {
			attemptSpan.LogError(err)

			cancelProc()
			if retryCount == 0 {
				return nil, errors.Wrap(err, "nsm client: Failed to connect")
			} else {
				attemptLogger.Errorf("nsm client: Failed to connect %v. Retry attempts: %v Delaying: %v", err, retryCount, retryDelay)
			}
			retryCount--
			attemptSpan.Finish()
			<-time.After(retryDelay)
			continue
		}
		break
	}
	span.Logger().Infof("Success connection")
	span.LogObject("connection", outgoingConnection)
	nsmc.OutgoingConnections = append(nsmc.OutgoingConnections, outgoingConnection)
	return outgoingConnection, nil
}

// Close will terminate a particular connection
func (nsmc *NsmClient) Close(ctx context.Context, outgoingConnection *connection.Connection) error {
	nsmc.Lock()
	defer nsmc.Unlock()

	span := spanhelper.FromContext(ctx, "Client.Close")
	defer span.Finish()
	span.LogObject("connection", outgoingConnection)

	_, err := nsmc.NsClient.Close(span.Context(), outgoingConnection)

	span.LogError(err)

	arr := nsmc.OutgoingConnections
	for i, c := range arr {
		if c == outgoingConnection {
			copy(arr[i:], arr[i+1:])
			arr[len(arr)-1] = nil
			arr = arr[:len(arr)-1]
		}
	}
	return err
}

// Destroy - Destroy stops the whole module
func (nsmc *NsmClient) Destroy(ctx context.Context) error {
	nsmc.Lock()
	defer nsmc.Unlock()

	span := spanhelper.FromContext(ctx, "Client.Destroy")
	defer span.Finish()

	err := nsmc.NsmConnection.Close()
	span.LogError(errors.Wrap(err, "failed to close opentracing context"))
	if nsmc.tracerCloser != nil {
		trErr := nsmc.tracerCloser.Close()
		if trErr != nil {
			logrus.Error(trErr)
		}
	}
	return err
}

// NewNSMClient creates the NsmClient
func NewNSMClient(ctx context.Context, configuration *common.NSConfiguration) (*NsmClient, error) {
	if configuration == nil {
		configuration = &common.NSConfiguration{}
	}

	client := &NsmClient{
		ClientNetworkService: configuration.ClientNetworkService,
		ClientLabels:         tools.ParseKVStringToMap(configuration.ClientLabels, ",", "="),
		NscInterfaceName:     configuration.NscInterfaceName,
	}

	client.tracerCloser = jaeger.InitJaeger("nsm-client")

	nsmConnection, err := common.NewNSMConnection(ctx, configuration)
	if err != nil {
		logrus.Errorf("Error: %v", err)
		return nil, err
	}

	client.NsmConnection = nsmConnection

	return client, nil
}
