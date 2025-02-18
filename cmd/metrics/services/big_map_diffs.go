package services

import (
	"context"
	"sync"

	"github.com/baking-bad/bcdhub/internal/config"
	"github.com/baking-bad/bcdhub/internal/logger"
	"github.com/baking-bad/bcdhub/internal/models"
)

// BigMapDiffHandler -
type BigMapDiffHandler struct {
	*config.Context
}

// NewBigMapDiffHandler -
func NewBigMapDiffHandler(ctx *config.Context) *BigMapDiffHandler {
	return &BigMapDiffHandler{ctx}
}

// Handle -
func (oh *BigMapDiffHandler) Handle(ctx context.Context, items []models.Model, wg *sync.WaitGroup) error {
	if len(items) == 0 {
		return nil
	}

	logger.Info().Msgf("%3d big map diffs are processed", len(items))

	return saveSearchModels(ctx, oh.Context, items)
}

// Chunk -
func (oh *BigMapDiffHandler) Chunk(lastID, size int64) ([]models.Model, error) {
	diffs, err := getDiffs(oh.StorageDB.DB, lastID, size)
	if err != nil {
		return nil, err
	}

	data := make([]models.Model, len(diffs))
	for i := range diffs {
		data[i] = &diffs[i]
	}
	return data, nil
}
