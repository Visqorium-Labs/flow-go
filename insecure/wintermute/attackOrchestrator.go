package wintermute

import (
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/insecure"
	"github.com/onflow/flow-go/insecure/adversary"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/component"
	"github.com/onflow/flow-go/module/irrecoverable"
	"github.com/onflow/flow-go/network"
)

type Orchestrator struct {
	component.Component
	logger       zerolog.Logger
	network      insecure.AttackNetwork
	corruptedIds flow.IdentityList
	allIds       flow.IdentityList // identity of all nodes in the network (including non-corrupted ones)
}

var _ insecure.AttackOrchestrator = &Orchestrator{}

// NewOrchestrator creates and returns a new Wintermute attack orchestrator.
func NewOrchestrator(logger zerolog.Logger, allIds flow.IdentityList, corruptedIds flow.IdentityList) *Orchestrator {
	o := &Orchestrator{
		logger:       logger,
		network:      adversary.NewAttackNetwork(corruptedIds, logger),
		corruptedIds: corruptedIds,
		allIds:       allIds,
	}

	cm := component.NewComponentManagerBuilder().
		AddWorker(func(ctx irrecoverable.SignalerContext, ready component.ReadyFunc) {
			o.start(ctx)

			ready()

			<-ctx.Done()
		}).Build()

	o.Component = cm

	return o
}

// start triggers the sub-modules of orchestrator.
func (o *Orchestrator) start(ctx irrecoverable.SignalerContext) {
	o.network.Start(ctx)
}

// HandleEventFromCorruptedNode implements logic of processing the events received from a corrupted node.
//
// In Corruptible Conduit Framework for BFT testing, corrupted nodes relay their outgoing events to
// the attacker instead of dispatching them to the network.
func (o *Orchestrator) HandleEventFromCorruptedNode(corruptedId flow.Identifier,
	channel network.Channel,
	event interface{},
	protocol insecure.Protocol,
	num uint32,
	targetIds ...flow.Identifier) error {

	panic("implement me")
}
