package process

import (
	"buf.build/gen/go/open-feature/flagd/grpc/go/flagd/sync/v1/syncv1grpc"
	v1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/flagd/sync/v1"
	"context"
	"fmt"
	"github.com/open-feature/flagd/core/pkg/logger"
	"github.com/open-feature/flagd/core/pkg/sync"
	grpccredential "github.com/open-feature/flagd/core/pkg/sync/grpc/credentials"
	of "github.com/open-feature/go-sdk/openfeature"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
	msync "sync"
	"time"
)

const (
	// Prefix for GRPC URL inputs. GRPC does not define a standard prefix. This prefix helps to differentiate remote
	// URLs for REST APIs (i.e - HTTP) from GRPC endpoints.
	Prefix          = "grpc://"
	PrefixSecure    = "grpcs://"
	SupportedScheme = "(envoy|dns|uds|xds)"
)

// type aliases for interfaces required by this component - needed for mock generation with gomock

type FlagSyncServiceClient interface {
	syncv1grpc.FlagSyncServiceClient
}
type FlagSyncServiceClientResponse interface {
	syncv1grpc.FlagSyncService_SyncFlagsClient
}

var once msync.Once

type Sync struct {
	GrpcDialOptionsOverride []grpc.DialOption
	CertPath                string
	CredentialBuilder       grpccredential.Builder
	Logger                  *logger.Logger
	ProviderID              string
	Secure                  bool
	Selector                string
	URI                     string
	MaxMsgSize              int

	client     FlagSyncServiceClient
	connection *grpc.ClientConn
	ready      bool
	events     chan SyncEvent
}

func (g *Sync) Init(ctx context.Context) error {
	var rpcCon *grpc.ClientConn
	var err error

	g.events = make(chan SyncEvent)

	if len(g.GrpcDialOptionsOverride) > 0 {
		g.Logger.Debug("GRPC DialOptions override provided")
		rpcCon, err = grpc.NewClient(g.URI, g.GrpcDialOptionsOverride...)
	} else {
		// Build dial options with enhanced features
		var dialOptions []grpc.DialOption

		// Transport credentials
		tCredentials, err := g.CredentialBuilder.Build(g.Secure, g.CertPath)
		if err != nil {
			err = fmt.Errorf("error building transport credentials: %w", err)
			g.Logger.Error(err.Error())
			return err
		}
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(tCredentials))

		// Call options
		var callOptions []grpc.CallOption
		if g.MaxMsgSize > 0 {
			callOptions = append(callOptions, grpc.MaxCallRecvMsgSize(g.MaxMsgSize))
			g.Logger.Info(fmt.Sprintf("setting max receive message size %d bytes", g.MaxMsgSize))
		}
		if len(callOptions) > 0 {
			dialOptions = append(dialOptions, grpc.WithDefaultCallOptions(callOptions...))
		}

		// Keepalive settings
		keepaliveParams := keepalive.ClientParameters{
			Time:                30 * time.Second, // Send ping every 30 seconds
			Timeout:             5 * time.Second,  // Wait 5 seconds for ping response
			PermitWithoutStream: true,             // Allow pings when no streams active
		}
		dialOptions = append(dialOptions, grpc.WithKeepaliveParams(keepaliveParams))

		// Create connection
		rpcCon, err = grpc.NewClient(g.URI, dialOptions...)
	}

	if err != nil {
		err := fmt.Errorf("error initiating grpc client connection: %w", err)
		g.Logger.Error(err.Error())
		return err
	}

	// Store connection for state tracking
	g.connection = rpcCon

	// Setup service client
	g.client = syncv1grpc.NewFlagSyncServiceClient(rpcCon)

	// Start connection state monitoring in background
	go g.monitorConnectionState(ctx)

	g.Logger.Info(fmt.Sprintf("gRPC client initialized for %s", g.URI))
	return nil
}

func (g *Sync) ReSync(ctx context.Context, dataSync chan<- sync.DataSync) error {
	res, err := g.client.FetchAllFlags(ctx, &v1.FetchAllFlagsRequest{ProviderId: g.ProviderID, Selector: g.Selector})
	if err != nil {
		err = fmt.Errorf("error fetching all flags: %w", err)
		g.Logger.Error(err.Error())
		return err
	}
	dataSync <- sync.DataSync{
		FlagData: res.GetFlagConfiguration(),
		Source:   g.URI,
	}
	return nil
}

func (g *Sync) IsReady() bool {
	return g.ready
}

func (g *Sync) Sync(ctx context.Context, dataSync chan<- sync.DataSync) error {
	for {
		g.Logger.Debug("creating sync stream...")

		// Create sync stream with wait-for-ready - let gRPC handle the connection waiting
		syncClient, err := g.client.SyncFlags(
			ctx,
			&v1.SyncFlagsRequest{
				ProviderId: g.ProviderID,
				Selector:   g.Selector,
			},
			grpc.WaitForReady(true), // gRPC will wait for connection to be ready
		)
		if err != nil {
			// Check if context is cancelled
			if ctx.Err() != nil {
				return ctx.Err()
			}

			g.Logger.Warn(fmt.Sprintf("failed to create sync stream: %v", err))

			// Brief pause before retry
			select {
			case <-time.After(time.Second):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		g.Logger.Info("sync stream established, starting to receive flags...")

		// Handle the stream - when it breaks, we'll create a new one
		err = g.handleFlagSync(syncClient, dataSync)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			g.Logger.Warn(fmt.Sprintf("stream closed: %v", err))
			// Loop will automatically create a new stream with wait-for-ready
		}
	}
}

// monitorConnectionState monitors connection state changes and logs errors
func (g *Sync) monitorConnectionState(ctx context.Context) {
	if g.connection == nil {
		return
	}

	currentState := g.connection.GetState()
	g.Logger.Debug(fmt.Sprintf("starting connection state monitoring, initial state: %s", currentState))

	for {
		// Wait for next state change
		if !g.connection.WaitForStateChange(ctx, currentState) {
			// Context cancelled, exit monitoring
			g.Logger.Debug("connection state monitoring stopped")
			return
		}

		newState := g.connection.GetState()
		g.Logger.Debug(fmt.Sprintf("connection state changed: %s -> %s", currentState, newState))

		// Log error states
		switch newState {
		case connectivity.TransientFailure:
			g.events <- SyncEvent{event: of.ProviderError}
			g.Logger.Error(fmt.Sprintf("gRPC connection entered TransientFailure state for %s", g.URI))
		case connectivity.Shutdown:
			g.Logger.Error(fmt.Sprintf("gRPC connection shutdown for %s", g.URI))
			//return // Exit monitoring on shutdown
		case connectivity.Ready:
			g.Logger.Info(fmt.Sprintf("gRPC connection ready for %s", g.URI))
		case connectivity.Idle:
			g.Logger.Debug(fmt.Sprintf("gRPC connection idle for %s", g.URI))
		case connectivity.Connecting:
			g.Logger.Debug(fmt.Sprintf("gRPC connection attempting to connect to %s", g.URI))
		}

		currentState = newState
	}
}

// handleFlagSync wraps the stream listening and push updates through dataSync channel
func (g *Sync) handleFlagSync(stream syncv1grpc.FlagSyncService_SyncFlagsClient, dataSync chan<- sync.DataSync) error {
	once.Do(func() {
		g.ready = true
	})

	// Stream message handling loop - receives each individual message from the stream
	for {
		data, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("error receiving payload from stream: %w", err)
		}

		dataSync <- sync.DataSync{
			FlagData:    data.FlagConfiguration,
			SyncContext: data.SyncContext,
			Source:      g.URI,
			Selector:    g.Selector,
		}

		g.Logger.Debug("received full configuration payload")
	}
}

func (g *Sync) Events() chan SyncEvent {
	return g.events
}
