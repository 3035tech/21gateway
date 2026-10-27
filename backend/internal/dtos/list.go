package dtos

import (
	"github.com/danielgtaylor/huma/v2"
	"reflect"
)

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

// ListQuery is shared by administrative list endpoints, not the legacy SMS API.
// Services must map Sort to an allowlist of model columns, never raw SQL.
type ListQuery struct {
	Active    OptionalBool  `query:"active"`
	CompanyID uint64        `query:"companyId"`
	State     string        `query:"state" enum:"pending,accepted,rejected,unknown"`
	Page      int           `query:"page" default:"1" minimum:"1"`
	PageSize  int           `query:"pageSize" default:"25" minimum:"1" maximum:"100"`
	Search    string        `query:"search" maxLength:"200"`
	Sort      string        `query:"sort" maxLength:"50"`
	Direction SortDirection `query:"direction" default:"asc" enum:"asc,desc"`
}

type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total" minimum:"0"`
	Page     int   `json:"page" minimum:"1"`
	PageSize int   `json:"pageSize" minimum:"1" maximum:"100"`
}

// Huma query parameters use ParamWrapper/ParamReactor to distinguish false from absent.
type OptionalBool struct {
	Value bool
	Set   bool
}

func (o OptionalBool) Schema(r huma.Registry) *huma.Schema {
	return huma.SchemaFromType(r, reflect.TypeFor[bool]())
}
func (o *OptionalBool) Receiver() reflect.Value    { return reflect.ValueOf(&o.Value).Elem() }
func (o *OptionalBool) OnParamSet(set bool, _ any) { o.Set = set }
