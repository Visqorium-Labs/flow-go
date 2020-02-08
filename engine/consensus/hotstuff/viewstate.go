package hotstuff

import (
	"github.com/dapperlabs/flow-go/model/flow"
)

type ViewState interface {
	// IsSelf returns if a given identity is myself
	IsSelf(id *flow.Identity) bool

	// IsSelfLeaderForView returns if myself is the leader at a given view
	IsSelfLeaderForView(view uint64) bool

	// GetIdentitiesForView returns all the staked nodes in my cluster at a certain block.
	// view specifies the view
	GetIdentitiesForBlockID(blockID flow.Identifier) (flow.IdentityList, error)

	// GetQCStakeThresholdForBlockID returns the stack threshold for building QC at a given block
	GetQCStakeThresholdForBlockID(blockID flow.Identifier) (uint64, error)

	// LeaderForView get the leader for a certain view
	LeaderForView(view uint64) *flow.Identity
}
