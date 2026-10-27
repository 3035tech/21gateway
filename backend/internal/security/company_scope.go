package security

import "context"

type companyScopeKey struct{}

type CompanyScope struct {
	Restricted bool
	IDs        []uint64
}

func WithCompanyScope(ctx context.Context, ids []uint64) context.Context {
	return context.WithValue(ctx, companyScopeKey{}, CompanyScope{Restricted: true, IDs: append([]uint64(nil), ids...)})
}

func Scope(ctx context.Context) CompanyScope {
	value, _ := ctx.Value(companyScopeKey{}).(CompanyScope)
	return value
}

func AllowsCompany(ctx context.Context, id uint64) bool {
	scope := Scope(ctx)
	if !scope.Restricted {
		return true
	}
	for _, allowed := range scope.IDs {
		if allowed == id {
			return true
		}
	}
	return false
}
