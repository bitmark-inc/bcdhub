package search

import (
	"fmt"
	"time"

	"github.com/baking-bad/bcdhub/internal/models"
	"github.com/baking-bad/bcdhub/internal/models/contract"
)

// Contract -
type Contract struct {
	Network   string    `json:"network"`
	Level     int64     `json:"level"`
	Timestamp time.Time `json:"timestamp"`
	Hash      string    `json:"hash"`

	Tags        []string `json:"tags,omitempty"`
	Hardcoded   []string `json:"hardcoded,omitempty"`
	FailStrings []string `json:"fail_strings,omitempty"`
	Annotations []string `json:"annotations,omitempty"`
	Entrypoints []string `json:"entrypoints,omitempty"`

	Address  string `json:"address"`
	Manager  string `json:"manager,omitempty"`
	Delegate string `json:"delegate,omitempty"`

	FoundBy       string `json:"found_by,omitempty"`
	Alias         string `json:"alias,omitempty"`
	DelegateAlias string `json:"delegate_alias,omitempty"`

	TxCount    *int64     `json:"tx_count,omitempty"`
	LastAction *time.Time `json:"last_action,omitempty"`
}

// GetID -
func (c *Contract) GetID() string {
	return fmt.Sprintf("%s_%s", c.Network, c.Address)
}

// GetIndex -
func (c Contract) GetIndex() string {
	return models.DocContracts
}

// GetScores -
func (c Contract) GetScores(search string) []string {
	return []string{
		"address^10",
		"alias^8",
		"tags^6",
		"fail_strings^1",
		"annotations^1",
		"hardcoded^1",
	}
}

// GetFields -
func (c Contract) GetFields() []string {
	return []string{
		"address",
		"alias",
		"tags",
		"fail_strings",
		"annotations",
		"hardcoded",
	}
}

// Parse  -
func (c Contract) Parse(highlight map[string][]string, data []byte) (*Item, error) {
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &Item{
		Type:       c.GetIndex(),
		Value:      c.Address,
		Body:       &c,
		Highlights: highlight,
		Network:    c.Network,
	}, nil
}

// Prepare -
func (c *Contract) Prepare(model models.Model) {
	cont, ok := model.(*contract.Contract)
	if !ok {
		return
	}

	script := cont.Alpha
	if cont.BabylonID > 0 {
		script = cont.Babylon
	}

	c.Address = cont.Account.Address
	c.Alias = cont.Account.Alias
	c.Annotations = script.Annotations
	c.Delegate = cont.Delegate.Address
	c.DelegateAlias = cont.Delegate.Alias
	c.Entrypoints = script.Entrypoints
	c.FailStrings = script.FailStrings
	c.Hardcoded = script.Hardcoded
	c.Hash = script.Hash
	c.Level = cont.Level
	c.Manager = cont.Manager.Address
	c.Network = cont.Network.String()
	c.Tags = cont.Tags.ToArray()
	c.Timestamp = cont.Timestamp.UTC()
	c.LastAction = &cont.LastAction
}
