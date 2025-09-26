package utils

type Metadata struct {
	CurrentPage  int64 `json:"currenPage,omitempty"`
	PageSize     int64 `json:"pageSize,omitempty"`
	FirstPage    int64 `json:"firstPage,omitempty"`
	LastPage     int64 `json:"lastPage,omitempty"`
	TotalRecords int64 `json:"totalRecords,omitempty"`
}

func CalculateMetadata(totalRecords, page, pageSize int64) *Metadata {
	if totalRecords == 0 {
		// Note that we return an empty Metadata struct if there are no records.
		return &Metadata{}
	}
	return &Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     (totalRecords + pageSize - 1) / pageSize,
		TotalRecords: totalRecords,
	}
}
