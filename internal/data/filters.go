package data

import (
	"slices"
	"strings"

	"github.com/ericksjp703/greenlight/internal/validator"
)

type Filters struct {
	Sort          string
	SorteableList []string
}

func ValidateFilters(v *validator.Validator, f Filters) {
	v.Check(!validator.In(f.Sort, f.SorteableList), "sort", "invalid parameter, must be one of: "+strings.Join(f.SorteableList, ", "))
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
