package transfer

import (
	"fmt"
	"strings"
	"time"

	"github.com/baking-bad/bcdhub/internal/models"
	"github.com/baking-bad/bcdhub/internal/models/dapp"
	"github.com/baking-bad/bcdhub/internal/models/transfer"
	"github.com/baking-bad/bcdhub/internal/models/types"
	"github.com/baking-bad/bcdhub/internal/postgres/core"
	"github.com/go-pg/pg/v10"
)

// Storage -
type Storage struct {
	*core.Postgres
}

// NewStorage -
func NewStorage(es *core.Postgres) *Storage {
	return &Storage{es}
}

// Get -
func (storage *Storage) Get(ctx transfer.GetContext) (po transfer.Pageable, err error) {
	po.Transfers = make([]transfer.Transfer, 0)
	query := storage.DB.Model(&po.Transfers)
	storage.buildGetContext(query, ctx, true)

	if err = query.Select(&po.Transfers); err != nil {
		return
	}

	received := len(po.Transfers)
	size := storage.GetPageSize(ctx.Size)
	if ctx.Offset == 0 && size > received {
		po.Total = int64(len(po.Transfers))
	} else {
		countQuery := storage.DB.Model().Table(models.DocTransfers)
		storage.buildGetContext(countQuery, ctx, false)
		count, err := countQuery.Count()
		if err != nil {
			return po, err
		}
		po.Total = int64(count)
	}

	if received > 0 {
		po.LastID = fmt.Sprintf("%d", po.Transfers[received-1].ID)
	}
	return po, nil
}

// GetAll -
func (storage *Storage) GetAll(network types.Network, level int64) ([]transfer.Transfer, error) {
	var transfers []transfer.Transfer
	err := storage.DB.Model(&transfers).
		Where("network = ?", network).
		Where("level = ?", level).
		Select(&transfers)
	return transfers, err
}

// GetTransfered -
func (storage *Storage) GetTransfered(network types.Network, contract string, tokenID uint64) (result float64, err error) {
	query := storage.DB.Model().Table(models.DocTransfers).ColumnExpr("COALESCE(SUM(amount), 0)").
		Where(`"to".address != '' AND "from".address != ''`).
		Relation("To").
		Relation("From")
	core.Token(network, contract, tokenID)(query)
	core.IsApplied(query)
	if err = query.Select(&result); err != nil {
		return
	}

	return
}

// GetToken24HoursVolume - returns token volume for last 24 hours
func (storage *Storage) GetToken24HoursVolume(network types.Network, contract string, initiators, entrypoints []string, tokenID uint64) (float64, error) {
	aDayAgo := time.Now().UTC().AddDate(0, 0, -1)

	var volume float64
	query := storage.DB.Model().Table(models.DocTransfers).
		ColumnExpr("COALESCE(SUM(amount), 0)").
		Where("timestamp > ?", aDayAgo)

	if len(entrypoints) > 0 {
		query.WhereIn("parent IN (?)", entrypoints)
	}
	if len(initiators) > 0 {
		query.Relation("Initiator").WhereIn("initiator.address IN (?)", initiators)
	}

	core.Token(network, contract, tokenID)(query)
	core.IsApplied(query)
	err := query.Select(&volume)

	return volume, err
}

const (
	tokenVolumeSeriesRequestTemplate = `
		with f as (
			select generate_series(
			date_trunc(?period, ?start_date),
			date_trunc(?period, now()),
			?interval ::interval
			) as val
		)
		select
			extract(epoch from f.val),
			sum(amount) as value
		from f
		left join transfers on date_trunc(?period, transfers.timestamp) = f.val where (transfers.from != transfers.to) and (status = 1) and token_id = ?token_id ?conditions
		group by 1
		order by date_part
	`
)

// TODO: realize
// GetTokenVolumeSeries -
func (storage *Storage) GetTokenVolumeSeries(network types.Network, period string, contracts []string, entrypoints []dapp.DAppContract, tokenID uint64) ([][]float64, error) {
	if err := core.ValidateHistogramPeriod(period); err != nil {
		return nil, err
	}

	conditions := make([]string, 0)
	if network != types.Empty {
		conditions = append(conditions, fmt.Sprintf("network = %d", network))
	}

	if len(contracts) > 0 {
		contractConditions := make([]string, len(contracts))
		for i := range contracts {
			contractConditions[i] = fmt.Sprintf("contract = '%s'", contracts[i])
		}
		conditions = append(conditions, strings.Join(contractConditions, " or "))
	}

	if len(entrypoints) > 0 {
		entrypointConditions := make([]string, 0)
		for _, e := range entrypoints {
			for j := range e.Entrypoint {
				entrypointConditions = append(entrypointConditions, fmt.Sprintf("(initiator = '%s' and parent = '%s')", e.Address, e.Entrypoint[j]))
			}
		}
		conditions = append(conditions, strings.Join(entrypointConditions, " or "))
	}

	stringConditions := strings.Join(conditions, ") and (")
	if len(stringConditions) > 0 {
		stringConditions = "and (" + stringConditions
		stringConditions += ")"
	}

	var resp []core.HistogramResponse
	if _, err := storage.DB.
		WithParam("token_id", tokenID).
		WithParam("period", period).
		WithParam("start_date", pg.Safe(core.GetHistogramInterval(period))).
		WithParam("interval", fmt.Sprintf("1 %s", period)).
		WithParam("conditions", pg.Safe(stringConditions)).
		Query(&resp, tokenVolumeSeriesRequestTemplate); err != nil {
		return nil, err
	}

	histogram := make([][]float64, 0, len(resp))
	for i := range resp {
		histogram = append(histogram, []float64{resp[i].DatePart * 1000, resp[i].Value})
	}
	return histogram, nil
}

const (
	calcBalanceRequest = `
	select (coalesce(value_to, 0) - coalesce(value_from, 0)) as balance, coalesce(t1.address, t2.address) as address, coalesce(t1.token_id, t2.token_id) as token_id from 
		(select sum(amount) as value_from, "from" as address, token_id from transfers where "from" is not null and contract = ?contract and network = ?network group by "from", token_id) t1
	full outer join 
		(select sum(amount) as value_to, "to" as address, token_id from transfers where "to" is not null and contract = ?contract and network = ?network group by "to", token_id) t2
		on t1.address = t2.address and t1.token_id = t2.token_id;`
)

// TODO: realize
// CalcBalances -
func (storage *Storage) CalcBalances(network types.Network, contract string) ([]transfer.Balance, error) {
	var balances []transfer.Balance
	_, err := storage.DB.
		WithParam("network", network).
		WithParam("contract", contract).
		Query(&balances, calcBalanceRequest)
	return balances, err
}
