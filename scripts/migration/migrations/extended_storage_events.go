package migrations

import (
	"context"
	"errors"

	"github.com/baking-bad/bcdhub/internal/bcd"
	"github.com/baking-bad/bcdhub/internal/bcd/ast"
	"github.com/baking-bad/bcdhub/internal/config"
	"github.com/baking-bad/bcdhub/internal/logger"
	"github.com/baking-bad/bcdhub/internal/models"
	"github.com/baking-bad/bcdhub/internal/models/bigmapdiff"
	"github.com/baking-bad/bcdhub/internal/models/contract_metadata"
	"github.com/baking-bad/bcdhub/internal/models/operation"
	"github.com/baking-bad/bcdhub/internal/models/transfer"
	"github.com/baking-bad/bcdhub/internal/models/types"
	"github.com/baking-bad/bcdhub/internal/noderpc"
	"github.com/baking-bad/bcdhub/internal/parsers/stacktrace"
	transferParsers "github.com/baking-bad/bcdhub/internal/parsers/transfer"
)

// ExtendedStorageEvents -
type ExtendedStorageEvents struct {
	contracts map[string]string
}

// Key -
func (m *ExtendedStorageEvents) Key() string {
	return "execute_extended_storage"
}

// Description -
func (m *ExtendedStorageEvents) Description() string {
	return "execute all extended storages"
}

// Do - migrate function
func (m *ExtendedStorageEvents) Do(ctx *config.Context) error {
	m.contracts = make(map[string]string)
	tzips, err := ctx.ContractMetadata.GetWithEvents(0)
	if err != nil {
		return err
	}

	logger.Info().Msgf("Found %d tzips", len(tzips))

	logger.Info().Msg("Execution events...")
	inserted := make([]models.Model, 0)
	deleted := make([]models.Model, 0)
	newTransfers := make([]*transfer.Transfer, 0)
	for i := range tzips {
		for _, event := range tzips[i].Events {
			for _, impl := range event.Implementations {
				if impl.MichelsonExtendedStorageEvent == nil || impl.MichelsonExtendedStorageEvent.Empty() {
					continue
				}
				logger.Info().Msgf("%s...", tzips[i].Address)

				protocol, err := ctx.Protocols.Get(tzips[i].Network, "", -1)
				if err != nil {
					if !ctx.Storage.IsRecordNotFound(err) {
						return err
					}
					protocol.Hash = bcd.GetCurrentProtocol()
					protocol.SymLink, err = bcd.GetProtoSymLink(protocol.Hash)
					if err != nil {
						return err
					}
				}
				rpc, err := ctx.GetRPC(tzips[i].Network)
				if err != nil {
					return err
				}

				operations, err := m.getOperations(ctx, tzips[i], impl)
				if err != nil {
					return err
				}

				if len(operations) == 0 {
					continue
				}

				script, err := ctx.Contracts.Script(tzips[i].Network, tzips[i].Address, protocol.SymLink)
				if err != nil {
					return err
				}

				for _, op := range operations {
					op.Script, err = script.Full()
					if err != nil {
						return err
					}
					op.AST, err = ast.NewScriptWithoutCode(op.Script)
					if err != nil {
						return err
					}

					st := stacktrace.New()
					if err := st.Fill(ctx.Operations, op); err != nil {
						return err
					}

					parser, err := transferParsers.NewParser(rpc, ctx.ContractMetadata, ctx.Blocks, ctx.TokenBalances, ctx.Accounts,
						transferParsers.WithNetwork(tzips[i].Network),
						transferParsers.WithGasLimit(protocol.Constants.HardGasLimitPerOperation),
						transferParsers.WithStackTrace(st),
					)
					if err != nil {
						return err
					}

					bmd, err := ctx.BigMapDiffs.GetForOperation(op.ID)
					if err != nil {
						if !ctx.Storage.IsRecordNotFound(err) {
							return err
						}
					}
					proto, err := ctx.Cache.ProtocolByID(operations[i].Network, operations[i].ProtocolID)
					if err != nil {
						return err
					}

					ptrsBmd := make([]*bigmapdiff.BigMapDiff, len(bmd))
					for i := range bmd {
						ptrsBmd[i] = &bmd[i]
					}

					if err := parser.Parse(ptrsBmd, proto.Hash, &op); err != nil {
						if errors.Is(err, noderpc.InvalidNodeResponse{}) {
							logger.Err(err)
							continue
						}
						return err
					}
					for _, t := range op.Transfers {
						old, err := ctx.Transfers.Get(transfer.GetContext{
							Network:     t.Network,
							TokenID:     &t.TokenID,
							OperationID: &op.ID,
						})
						if err != nil {
							return err
						}
						for j := range old.Transfers {
							deleted = append(deleted, &old.Transfers[j])
							m.contracts[old.Transfers[j].Contract] = old.Transfers[j].Network.String()
						}
						inserted = append(inserted, t)
						newTransfers = append(newTransfers, t)
						m.contracts[t.Contract] = t.Network.String()
					}
				}
			}
		}
	}
	logger.Info().Msgf("Delete %d transfers", len(deleted))
	if err := ctx.Storage.BulkDelete(context.Background(), deleted); err != nil {
		return err
	}

	logger.Info().Msgf("Found %d transfers", len(inserted))

	bu := transferParsers.UpdateTokenBalances(newTransfers)
	for i := range bu {
		inserted = append(inserted, bu[i])
	}

	return ctx.Storage.Save(context.Background(), inserted)
}

func (m *ExtendedStorageEvents) getOperations(ctx *config.Context, tzip contract_metadata.ContractMetadata, impl contract_metadata.EventImplementation) ([]operation.Operation, error) {
	operations := make([]operation.Operation, 0)

	for i := range impl.MichelsonExtendedStorageEvent.Entrypoints {
		ops, err := ctx.Operations.Get(map[string]interface{}{
			"operation.network":   tzip.Network,
			"destination.address": tzip.Address,
			"kind":                types.OperationKindTransaction,
			"status":              types.OperationStatusApplied,
			"entrypoint":          impl.MichelsonExtendedStorageEvent.Entrypoints[i],
		}, 0, false)
		if err != nil {
			return nil, err
		}
		operations = append(operations, ops...)
	}

	return operations, nil
}

// AffectedContracts -
func (m *ExtendedStorageEvents) AffectedContracts() map[string]string {
	return m.contracts
}
