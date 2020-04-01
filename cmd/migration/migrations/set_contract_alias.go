package migrations

import (
	"log"
	"time"

	"github.com/baking-bad/bcdhub/internal/elastic"
	"github.com/baking-bad/bcdhub/internal/metrics"
)

// SetContractAliasMigration - migration that set alias from db to contracts in choosen network
type SetContractAliasMigration struct {
	Network string
}

// Do - migrate function
func (m *SetContractAliasMigration) Do(ctx *Context) error {
	start := time.Now()
	h := metrics.New(ctx.ES, ctx.DB)

	filter := make(map[string]interface{})

	contracts, err := ctx.ES.GetContracts(filter)
	if err != nil {
		return err
	}

	aliases, err := ctx.DB.GetAliasesMap(m.Network)
	if err != nil {
		return err
	}

	for i := range contracts {
		h.SetContractAlias(aliases, &contracts[i])

		if _, err := ctx.ES.UpdateDoc(elastic.DocContracts, contracts[i].ID, contracts[i]); err != nil {
			return err
		}

		log.Printf("Done %d/%d", i, len(contracts))
	}

	log.Printf("Time spent: %v", time.Since(start))

	return nil
}
