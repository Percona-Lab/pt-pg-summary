// Package models contains the types for schema 'public'.
package models

// Code generated by xo. DO NOT EDIT.

import (
	"database/sql"
)

// IndexCacheHitRatio represents a row from '[custom index_cache_hit_ratio]'.
type IndexCacheHitRatio struct {
	Name  Unknown         // name
	Ratio sql.NullFloat64 // ratio
}

// GetIndexCacheHitRatio runs a custom query, returning results as IndexCacheHitRatio.
func GetIndexCacheHitRatio(db XODB) (*IndexCacheHitRatio, error) {
	var err error

	// sql query
	const sqlstr = `SELECT 'index hit rate' AS name, ` +
		`(sum(idx_blks_hit)) / sum(idx_blks_hit + idx_blks_read) AS ratio ` +
		`FROM pg_statio_user_indexes`

	// run query
	XOLog(sqlstr)
	var ichr IndexCacheHitRatio
	err = db.QueryRow(sqlstr).Scan(&ichr.Name, &ichr.Ratio)
	if err != nil {
		return nil, err
	}

	return &ichr, nil
}
