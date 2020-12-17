package metrics

import (
	"github.com/baking-bad/bcdhub/internal/database"
	"github.com/baking-bad/bcdhub/internal/models"
	"github.com/baking-bad/bcdhub/internal/models/bigmapdiff"
	"github.com/baking-bad/bcdhub/internal/models/block"
	"github.com/baking-bad/bcdhub/internal/models/contract"
	"github.com/baking-bad/bcdhub/internal/models/operation"
	"github.com/baking-bad/bcdhub/internal/models/protocol"
	"github.com/baking-bad/bcdhub/internal/models/schema"
	"github.com/baking-bad/bcdhub/internal/models/tokenbalance"
	"github.com/baking-bad/bcdhub/internal/models/tzip"
)

// Handler -
type Handler struct {
	Contracts     contract.Repository
	BigMapDiffs   bigmapdiff.Repository
	Blocks        block.Repository
	Protocol      protocol.Repository
	Operations    operation.Repository
	Schema        schema.Repository
	TokenBalances tokenbalance.Repository
	TZIP          tzip.Repository
	Storage       models.GeneralRepository
	Bulk          models.BulkRepository

	DB database.DB
}

// New -
func New(
	contracts contract.Repository,
	bmdRepo bigmapdiff.Repository,
	blocksRepo block.Repository,
	protocolRepo protocol.Repository,
	operations operation.Repository,
	schemaRepo schema.Repository,
	tbRepo tokenbalance.Repository,
	tzipRepo tzip.Repository,
	storage models.GeneralRepository,
	bulk models.BulkRepository,
	db database.DB,
) *Handler {
	return &Handler{contracts, bmdRepo, blocksRepo, protocolRepo, operations, schemaRepo, tbRepo, tzipRepo, storage, bulk, db}
}
