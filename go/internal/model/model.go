// Package model は DB の各テーブルの読み書きを行う. スキーマは PHP 版 (data/sql/tables.sql) のまま.
//
// 関数は *sqlx.DB と *sqlx.Tx のどちらでも使えるよう Q を受け取る.
// 見つからない場合は (nil, nil) を返す.
package model

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
)

type Q = sqlx.ExtContext

func get[T any](ctx context.Context, q Q, query string, args ...any) (*T, error) {
	var v T
	err := sqlx.GetContext(ctx, q, &v, query, args...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func list[T any](ctx context.Context, q Q, query string, args ...any) ([]T, error) {
	var v []T
	if err := sqlx.SelectContext(ctx, q, &v, query, args...); err != nil {
		return nil, err
	}
	return v, nil
}

// in は "IN (?)" を展開する.
func in(query string, args ...any) (string, []any, error) {
	return sqlx.In(query, args...)
}

// NullString は空でない文字列を NULL 可能な値にする. nil なら NULL.
func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
