package handlers

import (
	"fmt"
	"net/http"

	"github.com/baking-bad/bcdhub/internal/bcd"
	"github.com/baking-bad/bcdhub/internal/bcd/ast"
	"github.com/baking-bad/bcdhub/internal/bcd/consts"
	"github.com/baking-bad/bcdhub/internal/bcd/formatter"
	"github.com/baking-bad/bcdhub/internal/models/bigmapaction"
	"github.com/baking-bad/bcdhub/internal/models/bigmapdiff"
	"github.com/baking-bad/bcdhub/internal/models/types"
	"github.com/baking-bad/bcdhub/internal/noderpc"
	"github.com/gin-gonic/gin"
)

// GetBigMap godoc
// @Summary Get big map info by pointer
// @Description Get big map info by pointer
// @Tags bigmap
// @ID get-bigmap
// @Param network path string true "Network"
// @Param ptr path integer true "Big map pointer"
// @Accept  json
// @Produce  json
// @Success 200 {object} GetBigMapResponse
// @Failure 400 {object} Error
// @Failure 500 {object} Error
// @Router /v1/bigmap/{network}/{ptr} [get]
func (ctx *Context) GetBigMap(c *gin.Context) {
	var req getBigMapRequest
	if err := c.BindUri(&req); ctx.handleError(c, err, http.StatusBadRequest) {
		return
	}

	stats, err := ctx.BigMapDiffs.GetStats(req.NetworkID(), req.Ptr)
	if ctx.handleError(c, err, 0) {
		return
	}

	res := GetBigMapResponse{
		Network:    req.Network,
		Ptr:        req.Ptr,
		Address:    stats.Contract,
		TotalKeys:  uint(stats.Total),
		ActiveKeys: uint(stats.Active),
	}

	if stats.Total == 0 {
		actions, err := ctx.BigMapActions.Get(req.NetworkID(), req.Ptr)
		if ctx.handleError(c, err, 0) {
			return
		}
		if len(actions) > 0 {
			res.Address = actions[0].Address
		}
	} else {
		script, err := ctx.Contracts.ScriptPart(req.NetworkID(), res.Address, bcd.SymLinkBabylon, consts.STORAGE)
		if ctx.handleError(c, err, 0) {
			return
		}
		storage, err := ast.NewTypedAstFromBytes(script)
		if ctx.handleError(c, err, 0) {
			return
		}
		operation, err := ctx.Operations.Last(
			map[string]interface{}{
				"network":             req.NetworkID(),
				"destination.address": res.Address,
				"status":              types.OperationStatusApplied,
			}, 0)
		if ctx.handleError(c, err, 0) {
			return
		}
		proto, err := ctx.Cache.ProtocolByID(operation.Network, operation.ProtocolID)
		if ctx.handleError(c, err, 0) {
			return
		}

		var deffatedStorage []byte
		if proto.SymLink == bcd.SymLinkAlpha {
			rpc, err := ctx.GetRPC(req.NetworkID())
			if ctx.handleError(c, err, 0) {
				return
			}
			deffatedStorage, err = rpc.GetScriptStorageRaw(res.Address, 0)
			if ctx.handleError(c, err, 0) {
				return
			}
		} else {
			deffatedStorage = operation.DeffatedStorage
		}

		var data ast.UntypedAST
		if err := json.Unmarshal(deffatedStorage, &data); ctx.handleError(c, err, 0) {
			return
		}
		if err := storage.Settle(data); ctx.handleError(c, err, 0) {
			return
		}

		bigMap := storage.FindBigMapByPtr()
		for p, b := range bigMap {
			if p == req.Ptr {
				res.Typedef, _, err = b.Docs(ast.DocsFull)
				if ctx.handleError(c, err, 0) {
					return
				}
				break
			}
		}
	}

	alias, err := ctx.ContractMetadata.Get(req.NetworkID(), res.Address)
	if err != nil {
		if !ctx.Storage.IsRecordNotFound(err) {
			ctx.handleError(c, err, 0)
			return
		}
	} else {
		res.ContractAlias = alias.Name
	}

	c.SecureJSON(http.StatusOK, res)
}

// GetBigMapHistory godoc
// @Summary Get big map actions (alloc/copy/remove)
// @Description Get big map actions (alloc/copy/remove)
// @Tags bigmap
// @ID get-bigmap-history
// @Param network path string true "Network"
// @Param ptr path integer true "Big map pointer"
// @Accept  json
// @Produce  json
// @Success 200 {object} BigMapHistoryResponse
// @Success 204 {object} gin.H
// @Failure 400 {object} Error
// @Failure 500 {object} Error
// @Router /v1/bigmap/{network}/{ptr}/history [get]
func (ctx *Context) GetBigMapHistory(c *gin.Context) {
	var req getBigMapRequest
	if err := c.BindUri(&req); ctx.handleError(c, err, http.StatusBadRequest) {
		return
	}

	bm, err := ctx.BigMapActions.Get(req.NetworkID(), req.Ptr)
	if ctx.handleError(c, err, 0) {
		return
	}
	if bm == nil {
		c.SecureJSON(http.StatusNoContent, gin.H{})
		return
	}

	c.SecureJSON(http.StatusOK, prepareBigMapHistory(bm, req.Ptr))
}

// GetBigMapKeys godoc
// @Summary Get big map keys by pointer
// @Description Get big map keys by pointer
// @Tags bigmap
// @ID get-bigmap-keys
// @Param network path string true "Network"
// @Param ptr path integer true "Big map pointer"
// @Param q query string false "Search string"
// @Param offset query integer false "Offset"
// @Param size query integer false "Requested count" mininum(1) maximum(10)
// @Param max_level query integer false "Max level filter" minimum(0)
// @Param min_level query integer false "Min level filter" minimum(0)
// @Accept json
// @Produce json
// @Success 200 {array} BigMapResponseItem
// @Failure 400 {object} Error
// @Failure 500 {object} Error
// @Router /v1/bigmap/{network}/{ptr}/keys [get]
func (ctx *Context) GetBigMapKeys(c *gin.Context) {
	var req getBigMapRequest
	if err := c.BindUri(&req); ctx.handleError(c, err, http.StatusBadRequest) {
		return
	}

	var pageReq bigMapSearchRequest
	if err := c.BindQuery(&pageReq); ctx.handleError(c, err, http.StatusBadRequest) {
		return
	}

	states, err := ctx.BigMapDiffs.Keys(bigmapdiff.GetContext{
		Ptr:      &req.Ptr,
		Network:  req.NetworkID(),
		Query:    pageReq.Search,
		Size:     pageReq.Size,
		Offset:   pageReq.Offset,
		MaxLevel: pageReq.MaxLevel,
		MinLevel: pageReq.MinLevel,
	})
	if ctx.handleError(c, err, 0) {
		return
	}

	response, err := ctx.prepareBigMapKeys(states)
	if ctx.handleError(c, err, 0) {
		return
	}

	c.SecureJSON(http.StatusOK, response)
}

// GetBigMapByKeyHash godoc
// @Summary Get big map diffs by pointer and key hash
// @Description Get big map diffs by pointer and key hash
// @Tags bigmap
// @ID get-bigmap-keyhash
// @Param network path string true "Network"
// @Param ptr path integer true "Big map pointer"
// @Param key_hash path string true "Key hash in big map" minlength(54) maxlength(54)
// @Param offset query integer false "Offset"
// @Param size query integer false "Requested count" mininum(1) maximum(10)
// @Accept json
// @Produce json
// @Success 200 {object} BigMapDiffByKeyResponse
// @Success 204 {object} gin.H
// @Failure 400 {object} Error
// @Failure 500 {object} Error
// @Router /v1/bigmap/{network}/{ptr}/keys/{key_hash} [get]
func (ctx *Context) GetBigMapByKeyHash(c *gin.Context) {
	var req getBigMapByKeyHashRequest
	if err := c.BindUri(&req); ctx.handleError(c, err, http.StatusBadRequest) {
		return
	}

	var pageReq pageableRequest
	if err := c.BindQuery(&pageReq); ctx.handleError(c, err, http.StatusBadRequest) {
		return
	}

	bm, total, err := ctx.BigMapDiffs.GetByPtrAndKeyHash(req.Ptr, types.NewNetwork(req.Network), req.KeyHash, pageReq.Size, pageReq.Offset)
	if ctx.handleError(c, err, 0) {
		return
	}

	if total == 0 {
		c.SecureJSON(http.StatusNoContent, gin.H{})
		return
	}

	response, err := ctx.prepareBigMapItem(bm, req.KeyHash)
	if ctx.handleError(c, err, 0) {
		return
	}

	response.Total = total
	c.SecureJSON(http.StatusOK, response)
}

// GetBigMapDiffCount godoc
// @Summary Get big map diffs count info by pointer
// @Description Get big map diffs count info by pointer
// @Tags bigmap
// @ID get-bigmapdiff-count
// @Param network path string true "Network"
// @Param ptr path integer true "Big map pointer"
// @Accept  json
// @Produce  json
// @Success 200 {object} CountResponse
// @Failure 400 {object} Error
// @Failure 500 {object} Error
// @Router /v1/bigmap/{network}/{ptr}/count [get]
func (ctx *Context) GetBigMapDiffCount(c *gin.Context) {
	var req getBigMapRequest
	if err := c.BindUri(&req); ctx.handleError(c, err, http.StatusBadRequest) {
		return
	}

	count, err := ctx.BigMapDiffs.Count(req.NetworkID(), req.Ptr)
	if err != nil {
		if ctx.Storage.IsRecordNotFound(err) {
			c.SecureJSON(http.StatusOK, CountResponse{})
			return
		}
		ctx.handleError(c, err, 0)
		return
	}
	c.SecureJSON(http.StatusOK, CountResponse{count})
}

func (ctx *Context) prepareBigMapKeys(data []bigmapdiff.BigMapState) ([]BigMapResponseItem, error) {
	if len(data) == 0 {
		return []BigMapResponseItem{}, nil
	}

	bigMapType, err := ctx.getBigMapType(data[0].Network, data[0].Ptr)
	if err != nil {
		return nil, err
	}

	res := make([]BigMapResponseItem, len(data))
	for i := range data {
		key, value, keyString, err := prepareItem(data[i].Key, data[i].Value, bigMapType)
		if err != nil {
			return nil, err
		}

		res[i] = BigMapResponseItem{
			Item: BigMapItem{
				Key:       key,
				KeyHash:   data[i].KeyHash,
				KeyString: keyString,
				Level:     data[i].LastUpdateLevel,
				Value:     value,
				Timestamp: data[i].LastUpdateTime,
			},
			Count: data[i].Count,
		}
	}
	return res, nil
}

func (ctx *Context) prepareBigMapItem(data []bigmapdiff.BigMapDiff, keyHash string) (res BigMapDiffByKeyResponse, err error) {
	if len(data) == 0 {
		return
	}

	bigMapType, err := ctx.getBigMapType(data[0].Network, data[0].Ptr)
	if err != nil {
		return
	}

	var key, value interface{}
	values := make([]BigMapDiffItem, len(data))
	for i := range data {
		key, value, _, err = prepareItem(data[i].Key, data[i].Value, bigMapType)
		if err != nil {
			return
		}

		values[i] = BigMapDiffItem{
			Level:     data[i].Level,
			Value:     value,
			Timestamp: data[i].Timestamp,
		}

	}
	res.Values = values
	res.KeyHash = keyHash
	res.Key = key
	return
}

func prepareItem(itemKey, itemValue types.Bytes, bigMapType noderpc.BigMap) (key, value *ast.MiguelNode, keyString string, err error) {
	if itemKey != nil {
		keyType := ast.Copy(bigMapType.KeyType.Nodes[0])
		key, err = createMiguelForType(keyType, itemKey)
		if err != nil {
			return nil, nil, "", err
		}

		if key.Value != nil {
			switch t := key.Value.(type) {
			case string:
				keyString = t
			case int64:
				keyString = fmt.Sprintf("%d", t)
			default:
				keyString = fmt.Sprintf("%v", t)
			}
		} else {
			keyString, err = formatter.MichelineToMichelsonInline(string(itemKey))
			if err != nil {
				return nil, nil, "", err
			}
		}
	}

	if itemValue != nil {
		valueType := ast.Copy(bigMapType.ValueType.Nodes[0])
		valueMiguel, err := createMiguelForType(valueType, itemValue)
		if err != nil {
			return nil, nil, "", err
		}
		value = valueMiguel
	}

	return
}

func createMiguelForType(typ ast.Node, raw []byte) (*ast.MiguelNode, error) {
	var data ast.UntypedAST
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if err := typ.ParseValue(data[0]); err != nil {
		return nil, err
	}
	return typ.ToMiguel()
}

func prepareBigMapHistory(arr []bigmapaction.BigMapAction, ptr int64) BigMapHistoryResponse {
	if len(arr) == 0 {
		return BigMapHistoryResponse{}
	}
	response := BigMapHistoryResponse{
		Address: arr[0].Address,
		Network: arr[0].Network.String(),
		Ptr:     ptr,
		Items:   make([]BigMapHistoryItem, len(arr)),
	}

	for i := range arr {
		response.Items[i] = BigMapHistoryItem{
			Action:    arr[i].Action.String(),
			Timestamp: arr[i].Timestamp,
		}
		if arr[i].DestinationPtr != nil && *arr[i].DestinationPtr != ptr {
			response.Items[i].DestinationPtr = arr[i].DestinationPtr
		} else if arr[i].SourcePtr != nil && *arr[i].SourcePtr != ptr {
			response.Items[i].SourcePtr = arr[i].SourcePtr
		}
	}

	return response
}

func (ctx *Context) getBigMapType(network types.Network, ptr int64) (noderpc.BigMap, error) {
	rpc, err := ctx.GetRPC(network)
	if err != nil {
		return noderpc.BigMap{}, err
	}

	return rpc.GetBigMapType(ptr, 0)
}
