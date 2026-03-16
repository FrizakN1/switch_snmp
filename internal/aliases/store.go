package aliases

import (
	"encoding/json"
	"os"

	"snmp/internal/domain"
)

type jsonAliases struct {
	Aliases map[string]domain.MacAlias `json:"aliases"`
}

type Store struct {
	mac map[string]domain.MacAlias
}

func Load(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var a jsonAliases
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}

	return &Store{mac: a.Aliases}, nil
}

func (s *Store) Lookup(mac string) (domain.MacAlias, bool) {
	if s == nil {
		return domain.MacAlias{}, false
	}
	v, ok := s.mac[mac]
	return v, ok
}
