package data

import (
	"math"
	"slices"
	"strings"

	"github.com/ericksjp703/greenlight/internal/validator"
)

type Metadada struct {
	CurrentPage  int `json:"current_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
}

func CalculateMetadada(totalRecords, page, pageSize int) Metadada {
	if totalRecords == 0 {
		return Metadada{}
	}

	return Metadada{
		TotalRecords: totalRecords,
		CurrentPage:  page,
		FirstPage:    1,
		PageSize:     pageSize,
		LastPage:     int(math.Ceil(float64(totalRecords) / float64(pageSize))),
	}
}

type Filters struct {
	Page          int
	PageSize      int
	Sort          string
	SorteableList []string
}

func ValidateFilters(v *validator.Validator, f Filters) {
	// Check that the page and page_size parameters contain sensible values.
	v.Check(f.Page < 0, "page", "must be greater than zero")
	v.Check(f.Page > 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize < 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize > 100, "page_size", "must be a maximum of 100")

	v.Check(!validator.In(f.Sort, f.SorteableList), "sort", "must be one of: "+strings.Join(f.SorteableList, ", "))
}

func (f Filters) sortColumn() string {
	if slices.Contains(f.SorteableList, f.Sort) {
		return strings.TrimPrefix(f.Sort, "-")
	}

	panic("invalid sort value")
}

func (f Filters) sortOrder() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

func (f Filters) limit() int {
	return f.PageSize
}

func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}
