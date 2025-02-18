package migrations

import (
	"context"
	"fmt"

	"github.com/baking-bad/bcdhub/internal/config"
	"github.com/baking-bad/bcdhub/internal/models"
	"github.com/baking-bad/bcdhub/internal/models/contract"
	"github.com/baking-bad/bcdhub/internal/models/types"
	"github.com/baking-bad/bcdhub/internal/search"
	"github.com/go-pg/pg/v10"
)

// FixLostSearchContracts -
type FixLostSearchContracts struct {
	lastID int64
}

// Key -
func (m *FixLostSearchContracts) Key() string {
	return "fix_lost_search_contracts"
}

// Description -
func (m *FixLostSearchContracts) Description() string {
	return "fill `contracts` index in elasticsearch"
}

// Do - migrate function
func (m *FixLostSearchContracts) Do(ctx *config.Context) error {
	var err error
	contracts := make([]contract.Contract, 0)

	for m.lastID == 0 || len(contracts) == 1000 {
		fmt.Printf("last id = %d\r", m.lastID)
		contracts, err = m.getContracts(ctx.StorageDB.DB)
		if err != nil {
			return err
		}
		if err = m.saveSearchModels(ctx, contracts); err != nil {
			return err
		}
	}
	return nil
}

func (m *FixLostSearchContracts) getContracts(db *pg.DB) (resp []contract.Contract, err error) {
	query := db.Model().Table(models.DocContracts).Order("id asc")
	if m.lastID > 0 {
		query.Where("id > ?", m.lastID)
	}
	err = query.Limit(1000).Select(&resp)
	return
}

func (m *FixLostSearchContracts) saveSearchModels(internalContext *config.Context, contracts []contract.Contract) error {
	items := make([]models.Model, len(contracts))
	for i := range contracts {
		items[i] = &contracts[i]
		if m.lastID < contracts[i].ID {
			m.lastID = contracts[i].ID
		}
	}
	data := search.Prepare(items)

	for i := range data {
		if typ, ok := data[i].(*search.Contract); ok {
			typ.Alias = internalContext.Cache.Alias(types.NewNetwork(typ.Network), typ.Address)
			typ.DelegateAlias = internalContext.Cache.Alias(types.NewNetwork(typ.Network), typ.Delegate)
		}
	}

	return internalContext.Searcher.Save(context.Background(), data)
}
