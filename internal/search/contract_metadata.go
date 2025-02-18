package search

import (
	"fmt"
	"time"

	"github.com/baking-bad/bcdhub/internal/models"
	cm "github.com/baking-bad/bcdhub/internal/models/contract_metadata"
)

// Metadata -
type Metadata struct {
	Level       int64     `json:"level,omitempty"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
	Address     string    `json:"address"`
	Network     string    `json:"network"`
	Name        string    `json:"name,omitempty"`
	Description string    `json:"description,omitempty"`
	Homepage    string    `json:"homepage,omitempty"`
	Authors     []string  `json:"authors,omitempty"`
}

// GetID -
func (m *Metadata) GetID() string {
	return fmt.Sprintf("%s_%s", m.Network, m.Address)
}

// GetIndex -
func (m *Metadata) GetIndex() string {
	return models.DocContractMetadata
}

// GetScores -
func (m Metadata) GetScores(search string) []string {
	return []string{
		"name^8",
		"authors^6",
		"homepage^6",
		"description^5",
	}
}

// GetFields -
func (m Metadata) GetFields() []string {
	return []string{
		"name",
		"homepage",
		"description",
		"authors",
	}
}

// Parse  -
func (m Metadata) Parse(highlight map[string][]string, data []byte) (*Item, error) {
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &Item{
		Type:       m.GetIndex(),
		Value:      m.Address,
		Body:       &m,
		Highlights: highlight,
		Network:    m.Network,
	}, nil
}

// Prepare -
func (m *Metadata) Prepare(model models.Model) {
	t, ok := model.(*cm.ContractMetadata)
	if !ok {
		return
	}

	m.Address = t.Address
	m.Authors = t.Authors
	m.Description = t.Description
	m.Homepage = t.Homepage
	m.Level = t.Level
	m.Name = t.Name
	m.Network = t.Network.String()
	m.Timestamp = t.Timestamp.UTC()

}
