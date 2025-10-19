package meta

type Metadata struct {
	CurrentPage  int64 `json:"currentPage"`
	PageSize     int64 `json:"pageSize"`
	FirstPage    int64 `json:"firstPage"`
	LastPage     int64 `json:"lastPage"`
	TotalRecords int64 `json:"totalRecords"`
}

func CalculateMetadata(totalRecords, page, pageSize int64) *Metadata {
	if totalRecords == 0 {
		return &Metadata{
			CurrentPage:  1,
			PageSize:     pageSize,
			FirstPage:    1,
			LastPage:     1,
			TotalRecords: 0,
		}

	}
	return &Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     (totalRecords + pageSize - 1) / pageSize,
		TotalRecords: totalRecords,
	}
}
