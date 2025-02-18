package trees

import (
	"github.com/baking-bad/bcdhub/internal/bcd/ast"
	"github.com/baking-bad/bcdhub/internal/bcd/types"
	"github.com/baking-bad/bcdhub/internal/models/account"
	"github.com/baking-bad/bcdhub/internal/models/operation"
	"github.com/baking-bad/bcdhub/internal/models/transfer"
	modelTypes "github.com/baking-bad/bcdhub/internal/models/types"
	"github.com/shopspring/decimal"
)

// MakeFa2Transfers -
func MakeFa2Transfers(tree ast.Node, operation operation.Operation) ([]*transfer.Transfer, error) {
	if tree == nil {
		return nil, nil
	}
	transfers := make([]*transfer.Transfer, 0)
	list := tree.(*ast.List)
	for i := range list.Data {
		pair := list.Data[i].(*ast.Pair)
		from := pair.Args[0].GetValue().(string)
		toList := pair.Args[1].(*ast.List)
		for j := range toList.Data {
			var err error
			t := operation.EmptyTransfer()
			fromAddr, err := getAddress(from)
			if err != nil {
				return nil, err
			}
			t.From = account.Account{
				Network: operation.Network,
				Address: fromAddr,
				Type:    modelTypes.NewAccountType(fromAddr),
			}
			toPair := toList.Data[j].(*ast.Pair)
			to := toPair.Args[0].GetValue().(string)
			toAddr, err := getAddress(to)
			if err != nil {
				return nil, err
			}
			t.To = account.Account{
				Network: operation.Network,
				Address: toAddr,
				Type:    modelTypes.NewAccountType(toAddr),
			}
			tokenPair := toPair.Args[1].(*ast.Pair)
			t.TokenID = tokenPair.Args[0].GetValue().(*types.BigInt).Uint64()
			i := tokenPair.Args[1].GetValue().(*types.BigInt)
			t.Amount = decimal.NewFromBigInt(i.Int, 0)
			transfers = append(transfers, t)
		}
	}
	return transfers, nil
}
