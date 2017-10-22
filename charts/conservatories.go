package charts

type conservatory struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	Category   string  `json:"category"`
	Street     string  `json:"street"`
	City       string  `json:"city"`
	Department int     `json:"department"`
	Zip        string  `json:"zip"`
	Latitude   float64 `json:"lat"`
	Longitude  float64 `json:"lng"`
}

func listConservatories(page, pageSize int64, sortKey string) (int64, []*conservatory, error) {
	count, err := countConservatories()
	if err != nil {
		return 0, nil, err
	}

	list, err := readConservatories(page, pageSize, sortKey, true)

	return count, list, err
}
