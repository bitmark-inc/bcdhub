package handlers

import (
	"net/http"

	"github.com/aopoltorzhicky/bcdhub/internal/contractparser/consts"
	"github.com/aopoltorzhicky/bcdhub/internal/contractparser/meta"
	"github.com/aopoltorzhicky/bcdhub/internal/contractparser/miguel"
	"github.com/aopoltorzhicky/bcdhub/internal/elastic"
	"github.com/aopoltorzhicky/bcdhub/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// GetBigMap -
func (ctx *Context) GetBigMap(c *gin.Context) {
	var req getBigMapRequest
	if err := c.BindUri(&req); handleError(c, err, http.StatusBadRequest) {
		return
	}

	bm, err := ctx.ES.GetBigMap(req.Address, req.Ptr)
	if handleError(c, err, 0) {
		return
	}

	response, err := ctx.prepareBigMap(bm, req.Network, req.Address)
	if handleError(c, err, 0) {
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetBigMapByKeyHash -
func (ctx *Context) GetBigMapByKeyHash(c *gin.Context) {
	var req getBigMapByKeyHashRequest
	if err := c.BindUri(&req); handleError(c, err, http.StatusBadRequest) {
		return
	}

	bm, err := ctx.ES.GetBigMapDiffByPtrAndKeyHash(req.Address, req.Ptr, req.KeyHash)
	if handleError(c, err, 0) {
		return
	}

	response, err := ctx.prepareBigMapItem(bm, req.Network, req.Address)
	if handleError(c, err, 0) {
		return
	}

	c.JSON(http.StatusOK, response)
}

func (ctx *Context) prepareBigMap(data []elastic.BigMapDiff, network, address string) (res []BigMapResponseItem, err error) {
	alphaMeta, err := meta.GetMetadata(ctx.ES, address, network, "storage", consts.Hash1)
	if err != nil {
		return
	}

	babyMeta, err := meta.GetMetadata(ctx.ES, address, network, "storage", consts.HashBabylon)
	if err != nil {
		return
	}

	res = make([]BigMapResponseItem, len(data))
	for i := range data {
		var value interface{}
		if data[i].Data.Value != "" {
			val := gjson.Parse(data[i].Data.Value)
			metadata := babyMeta
			if network == consts.Mainnet && data[i].Data.Level < consts.LevelBabylon {
				metadata = alphaMeta
			}
			value, err = miguel.BigMapValueToMiguel(val, data[i].Data.BinPath, metadata)
			if err != nil {
				return
			}
		}

		res[i] = BigMapResponseItem{
			Item: BigMapItem{
				Key:     data[i].Data.Key,
				KeyHash: data[i].Data.KeyHash,
				Level:   data[i].Data.Level,
				Value:   value,
			},
			Count: data[i].Count,
		}
	}
	return
}

func (ctx *Context) prepareBigMapItem(data []models.BigMapDiff, network, address string) (res []BigMapItem, err error) {
	alphaMeta, err := meta.GetMetadata(ctx.ES, address, network, "storage", consts.Hash1)
	if err != nil {
		return
	}

	babyMeta, err := meta.GetMetadata(ctx.ES, address, network, "storage", consts.HashBabylon)
	if err != nil {
		return
	}

	res = make([]BigMapItem, len(data))
	for i := range data {
		var value interface{}
		if data[i].Value != "" {
			val := gjson.Parse(data[i].Value)
			metadata := babyMeta
			if network == consts.Mainnet && data[i].Level < consts.LevelBabylon {
				metadata = alphaMeta
			}
			value, err = miguel.BigMapValueToMiguel(val, data[i].BinPath, metadata)
			if err != nil {
				return
			}
		}

		res[i] = BigMapItem{
			Key:     data[i].Key,
			KeyHash: data[i].KeyHash,
			Level:   data[i].Level,
			Value:   value,
		}

	}
	return
}
