package profile

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

type Metadata struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Alias      string `json:"alias"`
	CutoffDate int    `json:"cutoffDate"`
	NPWP       string `json:"npwp"`
	TKUID      string `json:"tkuId"`
}

func LoadMetadataFromFile(rootDir string, profileID string) (Metadata, error) {
	path := profilepath.ProfileMetadataJSON(rootDir, profileID)
	b, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, err
	}

	var meta Metadata
	if err := json.Unmarshal(b, &meta); err != nil {
		return Metadata{}, err
	}

	meta.ID = strings.TrimSpace(meta.ID)
	meta.Name = strings.TrimSpace(meta.Name)
	meta.Alias = strings.TrimSpace(strings.ToUpper(meta.Alias))
	meta.NPWP = strings.TrimSpace(meta.NPWP)
	meta.TKUID = strings.TrimSpace(meta.TKUID)
	return meta, nil
}
