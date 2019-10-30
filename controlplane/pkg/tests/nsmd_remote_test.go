package tests

import (
	"context"
	"testing"
	"time"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/kernel"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection/mechanisms/vxlan"

	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connectioncontext"
	local "github.com/networkservicemesh/networkservicemesh/controlplane/api/local/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/local/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/pkg/model"
)

// Below only tests

func TestNSMDRequestClientRemoteNSMD(t *testing.T) {
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	srv2 := NewNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()

	srv.TestModel.AddForwarder(context.Background(), testForwarder1)

	srv2.TestModel.AddForwarder(context.Background(), testForwarder2)

	// Register in both
	nseReg := srv2.RegisterFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.TestModel.AddEndpoint(context.Background(), nseReg)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &local.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					DstIpRequired: true,
					SrcIpRequired: true,
				},
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*local.Mechanism{
			{
				Type: local.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					local.NetNsInodeKey:    "10",
					local.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for cross connections.
	cross_connections := srv2.serviceRegistry.testForwarderConnection.connections
	g.Expect(len(cross_connections)).To(Equal(1))
	logrus.Print("End of test")
}

func TestNSMDCloseCrossConnection(t *testing.T) {
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	srv2 := NewNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()
	srv.TestModel.AddForwarder(context.Background(), &model.Forwarder{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		LocalMechanisms: []*connection.Mechanism{
			&connection.Mechanism{
				Type: kernel.MECHANISM,
			},
		},
		RemoteMechanisms: []*connection.Mechanism{
			&connection.Mechanism{
				Type: vxlan.MECHANISM,
				Parameters: map[string]string{
					vxlan.VNI:   "1",
					vxlan.SrcIP: "10.1.1.1",
				},
			},
		},
		MechanismsConfigured: true,
	})

	srv2.TestModel.AddForwarder(context.Background(), &model.Forwarder{
		RegisteredName: "test_data_plane",
		SocketLocation: "tcp:some_addr",
		RemoteMechanisms: []*connection.Mechanism{
			&connection.Mechanism{
				Type: vxlan.MECHANISM,
				Parameters: map[string]string{
					vxlan.VNI:   "3",
					vxlan.SrcIP: "10.1.1.2",
				},
			},
		},
		MechanismsConfigured: true,
	})

	// Register in both
	nseReg := srv2.RegisterFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.TestModel.AddEndpoint(context.Background(), nseReg)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &local.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					DstIpRequired: true,
					SrcIpRequired: true,
				},
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*local.Mechanism{
			{
				Type: local.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					local.NetNsInodeKey:    "10",
					local.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	nsmResponse, err := nsmClient.Request(context.Background(), request)
	g.Expect(err).To(BeNil())
	g.Expect(nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for cross connections.
	cross_connection := srv.TestModel.GetClientConnection(nsmResponse.Id)
	g.Expect(cross_connection).ToNot(BeNil())

	destConnectionId := cross_connection.Xcon.GetRemoteDestination().GetId()

	cross_connection2 := srv2.TestModel.GetClientConnection(destConnectionId)
	g.Expect(cross_connection2).ToNot(BeNil())

	//Cross connection successfully created, check it closing
	_, err = nsmClient.Close(context.Background(), nsmResponse)
	g.Expect(err).To(BeNil())

	//We need to check that xcons have been removed from model
	cross_connection = srv.TestModel.GetClientConnection(nsmResponse.Id)
	g.Expect(cross_connection).To(BeNil())

	cross_connection2 = srv2.TestModel.GetClientConnection(destConnectionId)
	g.Expect(cross_connection2).To(BeNil())
}

func TestNSMDDelayRemoteMechanisms(t *testing.T) {
	g := NewWithT(t)

	storage := NewSharedStorage()
	srv := NewNSMDFullServer(Master, storage)
	srv2 := NewNSMDFullServer(Worker, storage)
	defer srv.Stop()
	defer srv2.Stop()

	srv.TestModel.AddForwarder(context.Background(), testForwarder1)

	testForwarder2_2 := &model.Forwarder{
		RegisteredName: "test_data_plane2",
		SocketLocation: "tcp:some_addr",
	}

	srv2.TestModel.AddForwarder(context.Background(), testForwarder2_2)

	// Register in both
	nseReg := srv2.RegisterFakeEndpoint("golden_network", "test", Worker)
	// Add to local endpoints for Server2
	srv2.TestModel.AddEndpoint(context.Background(), nseReg)

	// Now we could try to connect via Client API
	nsmClient, conn := srv.requestNSMConnection("nsm-1")
	defer conn.Close()

	request := &networkservice.NetworkServiceRequest{
		Connection: &local.Connection{
			NetworkService: "golden_network",
			Context: &connectioncontext.ConnectionContext{
				IpContext: &connectioncontext.IPContext{
					DstIpRequired: true,
					SrcIpRequired: true,
				},
			},
			Labels: make(map[string]string),
		},
		MechanismPreferences: []*local.Mechanism{
			{
				Type: local.MechanismType_KERNEL_INTERFACE,
				Parameters: map[string]string{
					local.NetNsInodeKey:    "10",
					local.InterfaceNameKey: "icmp-responder1",
				},
			},
		},
	}

	type Response struct {
		nsmResponse *local.Connection
		err         error
	}
	resultChan := make(chan *Response, 1)

	go func(ctx context.Context, req *networkservice.NetworkServiceRequest) {
		nsmResponse, err := nsmClient.Request(ctx, req)
		resultChan <- &Response{nsmResponse: nsmResponse, err: err}
	}(context.Background(), request)

	<-time.After(100 * time.Millisecond)

	testForwarder2_2.LocalMechanisms = testForwarder2.LocalMechanisms
	testForwarder2_2.RemoteMechanisms = testForwarder2.RemoteMechanisms
	testForwarder2_2.MechanismsConfigured = true
	srv2.TestModel.UpdateForwarder(context.Background(), testForwarder2_2)

	res := <-resultChan
	g.Expect(res.err).To(BeNil())
	g.Expect(res.nsmResponse.GetNetworkService()).To(Equal("golden_network"))

	// We need to check for crМфвук31oss connections.
	cross_connections := srv2.serviceRegistry.testForwarderConnection.connections
	g.Expect(len(cross_connections)).To(Equal(1))
	logrus.Print("End of test")
}
