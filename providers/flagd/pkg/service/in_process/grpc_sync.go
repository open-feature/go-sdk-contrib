package process

import (
	"buf.build/gen/go/open-feature/flagd/grpc/go/flagd/sync/v1/syncv1grpc"
	v1 "buf.build/gen/go/open-feature/flagd/protocolbuffers/go/flagd/sync/v1"
	"context"
	"fmt"
	"github.com/goccy/go-json"
	"github.com/open-feature/flagd/core/pkg/logger"
	"github.com/open-feature/flagd/core/pkg/sync"
	grpccredential "github.com/open-feature/flagd/core/pkg/sync/grpc/credentials"
	of "github.com/open-feature/go-sdk/openfeature"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	msync "sync"
	"time"
)

const (
	// Default timeouts and retry intervals
	defaultKeepaliveTime    = 30 * time.Second
	defaultKeepaliveTimeout = 5 * time.Second

	retryPolicy = `{
		  "methodConfig": [
			{
			  "name": [
				{
				  "service": "flagd.sync.v1.FlagSyncService"
				}
			  ],
			  "retryPolicy": {
				"MaxAttempts": 3,
				"InitialBackoff": "1s",
				"MaxBackoff": "5s",
				"BackoffMultiplier": 2.0,
				"RetryableStatusCodes": [
				  "CANCELLED",
				  "UNKNOWN",
				  "INVALID_ARGUMENT",
				  "NOT_FOUND",
				  "ALREADY_EXISTS",
				  "PERMISSION_DENIED",
				  "RESOURCE_EXHAUSTED",
				  "FAILED_PRECONDITION",
				  "ABORTED",
				  "OUT_OF_RANGE",
				  "UNIMPLEMENTED",
				  "INTERNAL",
				  "UNAVAILABLE",
				  "DATA_LOSS",
				  "UNAUTHENTICATED"
				]
			  }
			}
		  ]
		}`

	nonRetryableStatusCodes = `
		[
		  "PermissionDenied",
		  "Unauthenticated",
		]
	`
)

// Set of non-retryable gRPC status codes for faster lookup
var nonRetryableCodes map[string]struct{}

// Type aliases for interfaces required by this component - needed for mock generation with gomock
type FlagSyncServiceClient interface {
	syncv1grpc.FlagSyncServiceClient
}

type FlagSyncServiceClientResponse interface {
	syncv1grpc.FlagSyncService_SyncFlagsClient
}

// Sync implements gRPC-based flag synchronization with improved context cancellation and error handling
type Sync struct {
	// Configuration
	GrpcDialOptionsOverride []grpc.DialOption
	CertPath                string
	CredentialBuilder       grpccredential.Builder
	Logger                  *logger.Logger
	ProviderID              string
	Secure                  bool
	Selector                string
	URI                     string
	MaxMsgSize              int

	// Runtime state
	client           FlagSyncServiceClient
	connection       *grpc.ClientConn
	ready            bool
	events           chan SyncEvent
	shutdownComplete chan struct{}
	shutdownOnce     msync.Once
	initializer      msync.Once
}

// Init initializes the gRPC connection and starts background monitoring
func (g *Sync) Init(ctx context.Context) error {
	g.Logger.Info(fmt.Sprintf("initializing gRPC client for %s", g.URI))
	initNonRetryableStatusCodesSet()

	// Initialize channels
	g.shutdownComplete = make(chan struct{})
	g.events = make(chan SyncEvent, 10) // Buffered to prevent blocking

	// Establish gRPC connection
	conn, err := g.createConnection()
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	g.connection = conn
	g.client = syncv1grpc.NewFlagSyncServiceClient(conn)

	// Start connection state monitoring in background
	go g.monitorConnectionState(ctx)

	g.Logger.Info(fmt.Sprintf("gRPC client initialized successfully for %s", g.URI))
	return nil
}

// createConnection creates and configures the gRPC connection
func (g *Sync) createConnection() (*grpc.ClientConn, error) {
	if len(g.GrpcDialOptionsOverride) > 0 {
		g.Logger.Debug("using provided gRPC DialOptions override")
		return grpc.NewClient(g.URI, g.GrpcDialOptionsOverride...)
	}

	// Build standard dial options
	dialOptions, err := g.buildDialOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to build dial options: %w", err)
	}

	return grpc.NewClient(g.URI, dialOptions...)
}

// buildDialOptions constructs the standard gRPC dial options
func (g *Sync) buildDialOptions() ([]grpc.DialOption, error) {
	var dialOptions []grpc.DialOption

	// Transport credentials
	tCredentials, err := g.CredentialBuilder.Build(g.Secure, g.CertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build transport credentials: %w", err)
	}
	dialOptions = append(dialOptions, grpc.WithTransportCredentials(tCredentials))

	// Call options for message size
	if g.MaxMsgSize > 0 {
		callOptions := []grpc.CallOption{grpc.MaxCallRecvMsgSize(g.MaxMsgSize)}
		dialOptions = append(dialOptions, grpc.WithDefaultCallOptions(callOptions...))
		g.Logger.Info(fmt.Sprintf("setting max receive message size to %d bytes", g.MaxMsgSize))
	}

	// Keepalive settings for connection health
	keepaliveParams := keepalive.ClientParameters{
		Time:                defaultKeepaliveTime,
		Timeout:             defaultKeepaliveTimeout,
		PermitWithoutStream: true,
	}
	dialOptions = append(dialOptions, grpc.WithKeepaliveParams(keepaliveParams))

	dialOptions = append(dialOptions, grpc.WithDefaultServiceConfig(retryPolicy))

	return dialOptions, nil
}

// initNonRetryableStatusCodesSet initializes the set of non-retryable gRPC status codes for quick lookup
func initNonRetryableStatusCodesSet()  {
	var codes []string
	nonRetryableCodes = make(map[string]struct{})
	if err := json.Unmarshal([]byte(nonRetryableStatusCodes), &codes); err == nil {
		for _, code := range codes {
			nonRetryableCodes[code] = struct{}{}
		}
	}
}

// ReSync performs a one-time fetch of all flags
func (g *Sync) ReSync(ctx context.Context, dataSync chan<- sync.DataSync) error {
	g.Logger.Debug("performing ReSync - fetching all flags")

	res, err := g.client.FetchAllFlags(ctx, &v1.FetchAllFlagsRequest{
		ProviderId: g.ProviderID,
		Selector:   g.Selector,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch all flags: %w", err)
	}

	select {
	case dataSync <- sync.DataSync{
		FlagData: res.GetFlagConfiguration(),
		Source:   g.URI,
	}:
		g.Logger.Debug("ReSync completed successfully")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// IsReady returns whether the sync is ready to serve requests
func (g *Sync) IsReady() bool {
	return g.ready
}

// Sync starts the continuous flag synchronization process with improved context handling
func (g *Sync) Sync(ctx context.Context, dataSync chan<- sync.DataSync) error {
	g.Logger.Info("starting continuous flag synchronization")

	// Ensure shutdown completion is signaled when THIS method exits
	defer g.markShutdownComplete()

	for {
		// Check for cancellation before each iteration
		select {
		case <-ctx.Done():
			g.Logger.Info("sync stopped due to context cancellation")
			return ctx.Err()
		default:
			// Continue with sync logic
		}

		// Attempt to create sync stream
		err := g.performSyncCycle(ctx, dataSync)
		if err != nil {
			if ctx.Err() != nil {
				g.Logger.Info("sync cycle failed due to context cancellation")
				return ctx.Err()
			}

			// Check if error is a gRPC status error and if code is retryable
			st, ok := status.FromError(err)
			if ok {
				codeStr := st.Code().String()
				if _, found := nonRetryableCodes[codeStr]; found {
					g.Logger.Error(fmt.Sprintf("sync cycle failed with non-retryable code: %v", codeStr))
					return err
				}
			}

			g.Logger.Warn(fmt.Sprintf("sync cycle failed: %v, retrying...", err))
			g.sendEvent(ctx, SyncEvent{event: of.ProviderError})

			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}
}

// performSyncCycle handles a single sync cycle (create stream, handle messages, cleanup)
func (g *Sync) performSyncCycle(ctx context.Context, dataSync chan<- sync.DataSync) error {
	g.Logger.Debug("creating new sync stream")

	// Create sync stream with wait-for-ready to handle connection issues gracefully
	stream, err := g.client.SyncFlags(
		ctx,
		&v1.SyncFlagsRequest{
			ProviderId: g.ProviderID,
			Selector:   g.Selector,
		},
		grpc.WaitForReady(true),
	)
	if err != nil {
		return fmt.Errorf("failed to create sync stream: %w", err)
	}

	g.Logger.Info("sync stream established, starting to receive flags")

	// Handle the stream with proper context cancellation
	return g.handleFlagSync(ctx, stream, dataSync)
}

// handleFlagSync processes messages from the sync stream with proper context handling
func (g *Sync) handleFlagSync(ctx context.Context, stream syncv1grpc.FlagSyncService_SyncFlagsClient, dataSync chan<- sync.DataSync) error {
	// Mark as ready on first successful stream
	g.initializer.Do(func() {
		g.ready = true
		g.Logger.Info("sync service is now ready")
	})

	// Create channels for stream communication
	streamChan := make(chan *v1.SyncFlagsResponse, 1)
	errChan := make(chan error, 1)

	// Start goroutine to receive from stream
	go func() {
		defer close(streamChan)
		defer close(errChan)

		for {
			data, err := stream.Recv()
			if err != nil {
				select {
				case errChan <- err:
				case <-ctx.Done():
				}
				return
			}

			select {
			case streamChan <- data:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Main message handling loop with proper cancellation support
	for {
		select {
		case data, ok := <-streamChan:
			if !ok {
				return fmt.Errorf("stream channel closed")
			}

			if err := g.processFlagData(ctx, data, dataSync); err != nil {
				return err
			}

		case err := <-errChan:
			return fmt.Errorf("stream error: %w", err)

		case <-ctx.Done():
			g.Logger.Info("handleFlagSync stopped due to context cancellation")
			return ctx.Err()
		}
	}
}

// processFlagData handles individual flag configuration updates
func (g *Sync) processFlagData(ctx context.Context, data *v1.SyncFlagsResponse, dataSync chan<- sync.DataSync) error {
	syncData := sync.DataSync{
		FlagData:    data.FlagConfiguration,
		SyncContext: data.SyncContext,
		Source:      g.URI,
		Selector:    g.Selector,
	}

	select {
	case dataSync <- syncData:
		g.Logger.Debug("successfully processed flag configuration update")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// monitorConnectionState monitors gRPC connection state changes with improved cancellation handling
func (g *Sync) monitorConnectionState(ctx context.Context) {
	if g.connection == nil {
		g.Logger.Warn("no connection available for state monitoring")
		return
	}

	currentState := g.connection.GetState()
	g.Logger.Debug(fmt.Sprintf("starting connection state monitoring, initial state: %s", currentState))

	for {
		// Wait for state change with context support
		if !g.connection.WaitForStateChange(ctx, currentState) {
			g.Logger.Debug("connection state monitoring stopped due to context cancellation")
			return
		}

		// Check for cancellation
		select {
		case <-ctx.Done():
			g.Logger.Debug("connection state monitoring stopped due to context cancellation")
			return
		default:
		}

		newState := g.connection.GetState()
		g.Logger.Debug(fmt.Sprintf("connection state changed: %s -> %s", currentState, newState))

		// Handle state-specific logic
		g.handleConnectionState(ctx, newState)
		currentState = newState
	}
}

// handleConnectionState processes specific connection state changes
func (g *Sync) handleConnectionState(ctx context.Context, state connectivity.State) {
	switch state {
	case connectivity.TransientFailure:
		g.Logger.Error(fmt.Sprintf("gRPC connection entered TransientFailure state for %s", g.URI))
		g.sendEvent(ctx, SyncEvent{event: of.ProviderError})

	case connectivity.Shutdown:
		g.Logger.Error(fmt.Sprintf("gRPC connection shutdown for %s", g.URI))

	case connectivity.Ready:
		g.Logger.Info(fmt.Sprintf("gRPC connection ready for %s", g.URI))

	case connectivity.Idle:
		g.Logger.Debug(fmt.Sprintf("gRPC connection idle for %s", g.URI))

	case connectivity.Connecting:
		g.Logger.Debug(fmt.Sprintf("gRPC connection attempting to connect to %s", g.URI))
	}
}

// sendEvent safely sends events with cancellation support
func (g *Sync) sendEvent(ctx context.Context, event SyncEvent) {
	select {
	case g.events <- event:
		// Event sent successfully
	case <-ctx.Done():
		// Context cancelled, don't block
	default:
		// Channel full, log warning but don't block
		g.Logger.Warn("event channel full, dropping event")
	}
}

// markShutdownComplete signals that shutdown has completed
func (g *Sync) markShutdownComplete() {
	g.shutdownOnce.Do(func() {
		close(g.shutdownComplete)
		g.Logger.Debug("shutdown completion signaled")
	})
}

// Events returns the channel for sync events
func (g *Sync) Events() chan SyncEvent {
	return g.events
}

// Shutdown gracefully shuts down the sync service
func (g *Sync) Shutdown() error {
	g.Logger.Info("shutting down gRPC sync service")

	// Wait for shutdown completion with timeout
	select {
	case <-g.shutdownComplete:
		g.Logger.Info("sync operations completed gracefully")
	case <-time.After(5 * time.Second):
		g.Logger.Warn("shutdown timeout exceeded - forcing close")
	}

	// Close events channel
	if g.events != nil {
		close(g.events)
	}

	// Close gRPC connection
	if g.connection != nil {
		if err := g.connection.Close(); err != nil {
			g.Logger.Error(fmt.Sprintf("error closing gRPC connection: %v", err))
			return err
		}
	}

	g.Logger.Info("gRPC sync service shutdown completed successfully")
	return nil
}
