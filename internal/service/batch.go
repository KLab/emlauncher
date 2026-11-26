package service

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// maxDeletesPerApp は 1 回の実行でアプリごとに消す上限 (PHP 版と同じ). 残りは次の実行で消す.
const maxDeletesPerApp = 100

// DeleteOldPackages は, アプリごとに保護されていないパッケージを新しいものから keep 個だけ残して消す
// (古いパッケージの削除バッチ). 古いものから, アプリごとに 100 個まで消す. dryRun なら消さない.
// 経過を PHP 版 (batch/delete_old_packages.php) と同じ形式で w に書く.
func (s *Service) DeleteOldPackages(ctx context.Context, keep int, dryRun bool, w io.Writer) error {
	if keep < 0 {
		return fmt.Errorf("keep must not be negative: %d", keep)
	}
	appIDs, err := s.q.ListApplicationIDs(ctx)
	if err != nil {
		return err
	}
	suffix := ""
	if dryRun {
		suffix = " (DRYRUN)"
	}
	for _, appID := range appIDs {
		ids, err := s.q.ListUnprotectedPackageIDs(ctx, appID) // 新しい順
		if err != nil {
			return err
		}
		if len(ids) <= keep {
			continue
		}
		var targets []int64 // 古い順
		for i := len(ids) - 1; i >= keep && len(targets) < maxDeletesPerApp; i-- {
			targets = append(targets, ids[i])
		}
		fmt.Fprintf(w, "%s Application %d: %d packages%s\n", s.FormatTime(s.Now()), appID, len(targets), suffix)
		for _, id := range targets {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := s.deleteOldPackage(ctx, id, dryRun, w); err != nil {
				return err
			}
		}
	}
	return nil
}

// deleteOldPackage は 1 つのパッケージを消す. PHP 版と同じく, 処理の段階ごとに "." を出す
// (アプリとパッケージのロックで 3 つ, 行・ファイル・添付ファイルの削除で 3 つ).
// 一覧を取った後に他の処理で消された・保護されたものは消さずに "skipped" と出す.
func (s *Service) deleteOldPackage(ctx context.Context, id int64, dryRun bool, w io.Writer) error {
	fmt.Fprintf(w, "%s Delete package %d ", s.FormatTime(s.Now()), id)
	p, err := s.GetPackage(ctx, id)
	if errors.Is(err, ErrNotFound) || err == nil && p.Protect != 0 {
		fmt.Fprintln(w, "... skipped")
		return nil
	}
	if err != nil {
		return err
	}
	fmt.Fprint(w, "...")
	if !dryRun {
		// 保護されたかはロックを取った後に確かめ直す
		_, err := s.deletePackage(ctx, id, true)
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrProtected) {
			fmt.Fprintln(w, " skipped")
			return nil
		}
		if err != nil {
			fmt.Fprintln(w)
			return fmt.Errorf("delete package %d: %w", id, err)
		}
	}
	fmt.Fprintln(w, "... OK")
	return nil
}
