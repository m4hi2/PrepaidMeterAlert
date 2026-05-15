package datasources

import "github.com/m4hi2/MeterAlertBot/internal/database/models"

type Registry map[models.ProviderCode]DataFetcher

func (r Registry) Get(code models.ProviderCode) (DataFetcher, bool) {
	f, ok := r[code]
	return f, ok
}
