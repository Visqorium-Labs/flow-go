package node_builder

import (
	"context"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p-core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/onflow/flow/protobuf/go/flow/access"

	"github.com/onflow/flow-go/cmd"
	"github.com/onflow/flow-go/crypto"
	"github.com/onflow/flow-go/engine"
	"github.com/onflow/flow-go/engine/access/ingestion"
	pingeng "github.com/onflow/flow-go/engine/access/ping"
	"github.com/onflow/flow-go/engine/access/rpc"
	"github.com/onflow/flow-go/engine/access/rpc/backend"
	"github.com/onflow/flow-go/engine/common/requester"
	"github.com/onflow/flow-go/engine/common/splitter"
	synceng "github.com/onflow/flow-go/engine/common/synchronization"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/model/flow/filter"
	"github.com/onflow/flow-go/module"
	"github.com/onflow/flow-go/module/component"
	"github.com/onflow/flow-go/module/id"
	"github.com/onflow/flow-go/module/mempool/stdmap"
	"github.com/onflow/flow-go/module/metrics"
	"github.com/onflow/flow-go/module/metrics/unstaked"
	"github.com/onflow/flow-go/network"
	"github.com/onflow/flow-go/network/p2p"
	"github.com/onflow/flow-go/network/p2p/dns"
	"github.com/onflow/flow-go/network/topology"
	"github.com/onflow/flow-go/state/protocol/events/gadgets"
	"github.com/onflow/flow-go/utils/grpcutils"
)

// StakedAccessNodeBuilder builds a staked access node. The staked access node can optionally participate in the
// unstaked network publishing data for the unstaked access node downstream.
type StakedAccessNodeBuilder struct {
	*FlowAccessNodeBuilder
}

func NewStakedAccessNodeBuilder(anb *FlowAccessNodeBuilder) *StakedAccessNodeBuilder {
	return &StakedAccessNodeBuilder{
		FlowAccessNodeBuilder: anb,
	}
}

func (fnb *StakedAccessNodeBuilder) InitIDProviders() {
	fnb.Module("id providers", func(node *cmd.NodeConfig) error {

		idCache, err := p2p.NewProtocolStateIDCache(node.Logger, node.State, node.ProtocolEvents)
		if err != nil {
			return err
		}

		fnb.IdentityProvider = idCache

		fnb.SyncEngineParticipantsProviderFactory = func() id.IdentifierProvider {
			return id.NewIdentityFilterIdentifierProvider(
				filter.And(
					filter.HasRole(flow.RoleConsensus),
					filter.Not(filter.HasNodeID(node.Me.NodeID())),
					p2p.NotEjectedFilter,
				),
				idCache,
			)
		}

		fnb.IDTranslator = p2p.NewHierarchicalIDTranslator(idCache, p2p.NewUnstakedNetworkIDTranslator())

		return nil
	})
}

func (builder *StakedAccessNodeBuilder) Initialize() error {
	builder.InitComponentBuilder()
	builder.InitIDProviders()

	// if this is an access node that supports unstaked followers, enqueue the unstaked network
	if builder.supportsUnstakedFollower {
		builder.enqueueUnstakedNetworkInit()
	} else {
		// otherwise, enqueue the regular network
		builder.EnqueueNetworkInit()
	}

	builder.EnqueueMetricsServerInit()

	if err := builder.RegisterBadgerMetrics(); err != nil {
		return err
	}

	builder.EnqueueAdminServerInit()

	builder.EnqueueTracer()

	return nil
}

func (anb *StakedAccessNodeBuilder) Build() cmd.Node {
	anb.
		BuildConsensusFollower().
		Module("collection node client", func(node *cmd.NodeConfig) error {
			// collection node address is optional (if not specified, collection nodes will be chosen at random)
			if strings.TrimSpace(anb.rpcConf.CollectionAddr) == "" {
				node.Logger.Info().Msg("using a dynamic collection node address")
				return nil
			}

			node.Logger.Info().
				Str("collection_node", anb.rpcConf.CollectionAddr).
				Msg("using the static collection node address")

			collectionRPCConn, err := grpc.Dial(
				anb.rpcConf.CollectionAddr,
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
				grpc.WithInsecure(),
				backend.WithClientUnaryInterceptor(anb.rpcConf.CollectionClientTimeout))
			if err != nil {
				return err
			}
			anb.CollectionRPC = access.NewAccessAPIClient(collectionRPCConn)
			return nil
		}).
		Module("historical access node clients", func(node *cmd.NodeConfig) error {
			addrs := strings.Split(anb.rpcConf.HistoricalAccessAddrs, ",")
			for _, addr := range addrs {
				if strings.TrimSpace(addr) == "" {
					continue
				}
				node.Logger.Info().Str("access_nodes", addr).Msg("historical access node addresses")

				historicalAccessRPCConn, err := grpc.Dial(
					addr,
					grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(grpcutils.DefaultMaxMsgSize)),
					grpc.WithInsecure())
				if err != nil {
					return err
				}
				anb.HistoricalAccessRPCs = append(anb.HistoricalAccessRPCs, access.NewAccessAPIClient(historicalAccessRPCConn))
			}
			return nil
		}).
		Module("transaction timing mempools", func(node *cmd.NodeConfig) error {
			var err error
			anb.TransactionTimings, err = stdmap.NewTransactionTimings(1500 * 300) // assume 1500 TPS * 300 seconds
			if err != nil {
				return err
			}

			anb.CollectionsToMarkFinalized, err = stdmap.NewTimes(50 * 300) // assume 50 collection nodes * 300 seconds
			if err != nil {
				return err
			}

			anb.CollectionsToMarkExecuted, err = stdmap.NewTimes(50 * 300) // assume 50 collection nodes * 300 seconds
			if err != nil {
				return err
			}

			anb.BlocksToMarkExecuted, err = stdmap.NewTimes(1 * 300) // assume 1 block per second * 300 seconds
			return err
		}).
		Module("transaction metrics", func(node *cmd.NodeConfig) error {
			anb.TransactionMetrics = metrics.NewTransactionCollector(anb.TransactionTimings, node.Logger, anb.logTxTimeToFinalized,
				anb.logTxTimeToExecuted, anb.logTxTimeToFinalizedExecuted)
			return nil
		}).
		Module("ping metrics", func(node *cmd.NodeConfig) error {
			anb.PingMetrics = metrics.NewPingCollector()
			return nil
		}).
		Module("server certificate", func(node *cmd.NodeConfig) error {
			// generate the server certificate that will be served by the GRPC server
			x509Certificate, err := grpcutils.X509Certificate(node.NetworkKey)
			if err != nil {
				return err
			}
			tlsConfig := grpcutils.DefaultServerTLSConfig(x509Certificate)
			anb.rpcConf.TransportCredentials = credentials.NewTLS(tlsConfig)
			return nil
		}).
		Component("RPC engine", func(node *cmd.NodeConfig, lookup component.LookupFunc) (module.ReadyDoneAware, error) {
			anb.RpcEng = rpc.New(
				node.Logger,
				node.State,
				anb.rpcConf,
				anb.CollectionRPC,
				anb.HistoricalAccessRPCs,
				node.Storage.Blocks,
				node.Storage.Headers,
				node.Storage.Collections,
				node.Storage.Transactions,
				node.Storage.Receipts,
				node.Storage.Results,
				node.RootChainID,
				anb.TransactionMetrics,
				anb.collectionGRPCPort,
				anb.executionGRPCPort,
				anb.retryEnabled,
				anb.rpcMetricsEnabled,
				anb.apiRatelimits,
				anb.apiBurstlimits,
			)
			return anb.RpcEng, nil
		}).
		Component("ingestion engine", func(node *cmd.NodeConfig, lookup component.LookupFunc) (module.ReadyDoneAware, error) {
			var err error

			anb.RequestEng, err = requester.New(
				node.Logger,
				node.Metrics.Engine,
				node.Network,
				node.Me,
				node.State,
				engine.RequestCollections,
				filter.HasRole(flow.RoleCollection),
				func() flow.Entity { return &flow.Collection{} },
			)
			if err != nil {
				return nil, fmt.Errorf("could not create requester engine: %w", err)
			}

			anb.IngestEng, err = ingestion.New(node.Logger, node.Network, node.State, node.Me, anb.RequestEng, node.Storage.Blocks, node.Storage.Headers, node.Storage.Collections, node.Storage.Transactions, node.Storage.Results, node.Storage.Receipts, anb.TransactionMetrics,
				anb.CollectionsToMarkFinalized, anb.CollectionsToMarkExecuted, anb.BlocksToMarkExecuted, anb.RpcEng)
			if err != nil {
				return nil, err
			}
			anb.RequestEng.WithHandle(anb.IngestEng.OnCollection)
			anb.FinalizationDistributor.AddConsumer(anb.IngestEng)

			return anb.IngestEng, nil
		}).
		Component("requester engine", func(node *cmd.NodeConfig, lookup component.LookupFunc) (module.ReadyDoneAware, error) {
			// We initialize the requester engine inside the ingestion engine due to the mutual dependency. However, in
			// order for it to properly start and shut down, we should still return it as its own engine here, so it can
			// be handled by the scaffold.
			return anb.RequestEng, nil
		})

	if anb.supportsUnstakedFollower {
		var unstakedNetworkConduit network.Conduit
		var proxyEngine *splitter.Engine

		anb.
			Component("unstaked sync request proxy", func(node *cmd.NodeConfig, lookup component.LookupFunc) (module.ReadyDoneAware, error) {
				proxyEngine = splitter.New(node.Logger, engine.PublicSyncCommittee)

				// register the proxy engine with the unstaked network
				var err error
				unstakedNetworkConduit, err = node.Network.Register(engine.PublicSyncCommittee, proxyEngine)
				if err != nil {
					return nil, fmt.Errorf("could not register unstaked sync request proxy: %w", err)
				}

				return proxyEngine, nil
			}).
			Component("unstaked sync request handler", func(node *cmd.NodeConfig, lookup component.LookupFunc) (module.ReadyDoneAware, error) {
				syncRequestHandler := synceng.NewRequestHandlerEngine(
					node.Logger.With().Bool("unstaked", true).Logger(),
					unstaked.NewUnstakedEngineCollector(node.Metrics.Engine),
					unstakedNetworkConduit,
					node.Me,
					node.Storage.Blocks,
					anb.SyncCore,
					anb.FinalizedHeader,
					// don't queue missing heights from unstaked nodes
					// since we are always more up-to-date than them
					false,
				)

				// register the sync request handler with the proxy engine
				proxyEngine.RegisterEngine(syncRequestHandler)

				return syncRequestHandler, nil
			})
	}

	anb.Component("ping engine", func(node *cmd.NodeConfig, lookup component.LookupFunc) (module.ReadyDoneAware, error) {
		ping, err := pingeng.New(
			node.Logger,
			node.State,
			node.Me,
			anb.PingMetrics,
			anb.pingEnabled,
			node.Middleware,
			anb.nodeInfoFile,
		)
		if err != nil {
			return nil, fmt.Errorf("could not create ping engine: %w", err)
		}
		return ping, nil
	})

	return anb.FlowAccessNodeBuilder.Build()
}

// enqueueUnstakedNetworkInit enqueues the unstaked network component initialized for the staked node
func (builder *StakedAccessNodeBuilder) enqueueUnstakedNetworkInit() {

	builder.Component("unstaked network", func(node *cmd.NodeConfig, lookup component.LookupFunc) (module.ReadyDoneAware, error) {

		libP2PFactory := builder.initLibP2PFactory(builder.NodeID, builder.NodeConfig.NetworkKey)

		msgValidators := unstakedNetworkMsgValidators(node.Logger, node.IdentityProvider, builder.NodeID)

		middleware := builder.initMiddleware(builder.NodeID, node.Metrics.Network, libP2PFactory, msgValidators...)

		// topology returns empty list since peers are not known upfront
		top := topology.EmptyListTopology{}

		network, err := builder.initNetwork(builder.Me, node.Metrics.Network, middleware, top)
		if err != nil {
			return nil, err
		}

		builder.Network = network
		builder.Middleware = middleware

		idEvents := gadgets.NewIdentityDeltas(builder.Middleware.UpdateNodeAddresses)
		builder.ProtocolEvents.AddConsumer(idEvents)

		node.Logger.Info().Msgf("network will run on address: %s", builder.BindAddr)
		return builder.Network, nil
	})
}

// initLibP2PFactory creates the LibP2P factory function for the given node ID and network key.
// The factory function is later passed into the initMiddleware function to eventually instantiate the p2p.LibP2PNode instance
// The LibP2P host is created with the following options:
// 		DHT as server
// 		The address from the node config or the specified bind address as the listen address
// 		The passed in private key as the libp2p key
//		No connection gater
// 		Default Flow libp2p pubsub options
func (builder *StakedAccessNodeBuilder) initLibP2PFactory(nodeID flow.Identifier, networkKey crypto.PrivateKey) p2p.LibP2PFactoryFunc {

	// The staked nodes act as the DHT servers
	dhtOptions := []dht.Option{p2p.AsServer(true)}

	psOpts := append(p2p.DefaultPubsubOptions(p2p.DefaultMaxPubSubMsgSize),
		func(_ context.Context, h host.Host) (pubsub.Option, error) {
			return pubsub.WithSubscriptionFilter(p2p.NewRoleBasedFilter(
				h.ID(), builder.IdentityProvider,
			)), nil
		})

	myAddr := builder.NodeConfig.Me.Address()
	if builder.BaseConfig.BindAddr != cmd.NotSet {
		myAddr = builder.BaseConfig.BindAddr
	}

	connManager := p2p.NewConnManager(builder.Logger, builder.Metrics.Network, p2p.TrackUnstakedConnections(builder.IdentityProvider))

	resolver := dns.NewResolver(builder.Metrics.Network, dns.WithTTL(builder.BaseConfig.DNSCacheTTL))

	return func(ctx context.Context) (*p2p.Node, error) {
		libp2pNode, err := p2p.NewDefaultLibP2PNodeBuilder(nodeID, myAddr, networkKey).
			SetSporkID(builder.SporkID).
			// no connection gater
			SetConnectionManager(connManager).
			// act as a DHT server
			SetDHTOptions(dhtOptions...).
			SetPubsubOptions(psOpts...).
			SetLogger(builder.Logger).
			SetResolver(resolver).
			Build(ctx)
		if err != nil {
			return nil, err
		}
		builder.LibP2PNode = libp2pNode
		return builder.LibP2PNode, nil
	}
}

// initMiddleware creates the network.Middleware implementation with the libp2p factory function, metrics, peer update
// interval, and validators. The network.Middleware is then passed into the initNetwork function.
func (builder *StakedAccessNodeBuilder) initMiddleware(nodeID flow.Identifier,
	networkMetrics module.NetworkMetrics,
	factoryFunc p2p.LibP2PFactoryFunc,
	validators ...network.MessageValidator) network.Middleware {

	// disable connection pruning for the staked AN which supports the unstaked AN
	peerManagerFactory := p2p.PeerManagerFactory([]p2p.Option{p2p.WithInterval(builder.PeerUpdateInterval)}, p2p.WithConnectionPruning(false))

	builder.Middleware = p2p.NewMiddleware(
		builder.Logger,
		factoryFunc,
		nodeID,
		networkMetrics,
		builder.SporkID,
		p2p.DefaultUnicastTimeout,
		builder.IDTranslator,
		p2p.WithMessageValidators(validators...),
		p2p.WithPeerManager(peerManagerFactory),
		p2p.WithConnectionGating(false),
		// use default identifier provider
	)

	return builder.Middleware
}
