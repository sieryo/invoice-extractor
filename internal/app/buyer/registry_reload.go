package buyer

import (
	"log/slog"

	"github.com/sieryo/invoice-extractor/internal/infra/storage"
)

func (r *Registry) ReloadFromCSV(
	store *storage.BuyerCSVStore,
	csvPath string,
) error {
	buyers, err := store.Load(csvPath)
	if err != nil {
		r.loaded = false
		return err
	}

	r.Load(buyers)

	slog.Info("buyer registry reloaded",
		"count", len(buyers),
		"path", csvPath,
	)

	r.loaded = true

	return nil
}
