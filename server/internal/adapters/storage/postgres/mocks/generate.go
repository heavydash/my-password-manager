//go:build generate
// +build generate

package mocks

//go:generate mockgen -package=mocks -destination=dbpool_mock.go -source=../../../ports/db.go DBPool
