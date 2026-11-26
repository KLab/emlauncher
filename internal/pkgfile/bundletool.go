package pkgfile

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Bundletool は bundletool (Java) で Android App Bundle から universal APK を作る.
// 設定 apkfile に対応する.
type Bundletool struct {
	Java     []string // java コマンドとオプション (例: java -Xmx2048m)
	Jar      string
	Keystore string
	KSPass   string // "pass:xxx" または "file:/path"
	KeyAlias string
	KeyPass  string
	AAPT2    string // 空なら bundletool 内蔵のもの
	Keytool  string // keytool コマンド. 空なら "keytool"
	// MaxConcurrency は同時に動かす Java のプロセスの数の上限 (メモリを大きく使うため). 0 なら上限なし.
	MaxConcurrency int
	// Timeout は bundletool の実行時間の上限. 0 なら上限なし.
	Timeout time.Duration

	slotsOnce sync.Once
	slots     chan struct{}
}

const (
	// keytoolTimeout は keytool (キーストアの表示) を待つ上限.
	keytoolTimeout = time.Minute
)

// acquireJava は Java のプロセスを動かす枠を取る (MaxConcurrency が 0 なら取らない). 返す関数で返す.
func (b *Bundletool) acquireJava(ctx context.Context) (func(), error) {
	b.slotsOnce.Do(func() {
		if b.MaxConcurrency > 0 {
			b.slots = make(chan struct{}, b.MaxConcurrency)
		}
	})
	if b.slots == nil {
		return func() {}, nil
	}
	select {
	case b.slots <- struct{}{}:
		return func() { <-b.slots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// passwordArg は "pass:xxx" のパスワードを dir の中のファイル (0600) に書き, "file:{path}" にする
// (パスワードをコマンドラインに出さないため). "file:..." などはそのまま.
func passwordArg(dir, name, v string) (string, error) {
	pass, ok := strings.CutPrefix(v, "pass:")
	if !ok {
		return v, nil
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(pass), 0o600); err != nil {
		return "", err
	}
	return "file:" + p, nil
}

// UniversalAPK は aabPath から universal APK を作り, outPath に書く.
func (b *Bundletool) UniversalAPK(ctx context.Context, aabPath, outPath string) error {
	if len(b.Java) == 0 || b.Jar == "" {
		return errors.New("bundletool: apkfile.java / apkfile.bundletool is not configured")
	}
	release, err := b.acquireJava(ctx)
	if err != nil {
		return err
	}
	defer release()
	if b.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, b.Timeout)
		defer cancel()
	}

	// MkdirTemp は 0700 で作る. パスワードのファイルもここに置く
	dir, err := os.MkdirTemp("", "emlauncher-apks-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	apks := filepath.Join(dir, "out.apks")
	ksPass, err := passwordArg(dir, "ks-pass", b.KSPass)
	if err != nil {
		return err
	}
	keyPass, err := passwordArg(dir, "key-pass", b.KeyPass)
	if err != nil {
		return err
	}

	args := append(append([]string{}, b.Java[1:]...),
		"-jar", b.Jar, "build-apks", "--mode=universal",
		"--bundle="+aabPath,
		"--output="+apks,
		"--ks="+b.Keystore,
		"--ks-pass="+ksPass,
		"--ks-key-alias="+b.KeyAlias,
		"--key-pass="+keyPass,
	)
	if b.AAPT2 != "" {
		args = append(args, "--aapt2="+b.AAPT2)
	}
	cmd := exec.CommandContext(ctx, b.Java[0], args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("bundletool: %w: %s", err, bytes.TrimSpace(out))
	}
	return extractUniversalAPK(apks, outPath)
}

func extractUniversalAPK(apks, outPath string) error {
	zr, err := zip.OpenReader(apks)
	if err != nil {
		return fmt.Errorf("bundletool: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "universal.apk" {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return fmt.Errorf("bundletool: universal.apk: %w", err)
		}
		defer r.Close()
		w, err := os.Create(outPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, r); err != nil {
			w.Close()
			return fmt.Errorf("bundletool: universal.apk: %w", err)
		}
		return w.Close()
	}
	return errors.New("bundletool: universal.apk not found in the output")
}

// KeystoreInfo は署名に使うキーストアの情報 (keytool -list -v の出力) を返す.
func (b *Bundletool) KeystoreInfo(ctx context.Context) (string, error) {
	keytool := b.Keytool
	if keytool == "" {
		keytool = "keytool"
	}
	release, err := b.acquireJava(ctx)
	if err != nil {
		return "", err
	}
	defer release()
	ctx, cancel := context.WithTimeout(ctx, keytoolTimeout)
	defer cancel()
	args := []string{"-list", "-v", "-keystore", b.Keystore, "-alias", b.KeyAlias}
	var env []string
	switch kind, value, _ := strings.Cut(b.KSPass, ":"); kind {
	case "pass":
		// コマンドラインに出さないよう環境変数で渡す
		args = append(args, "-storepass:env", "EMLAUNCHER_STOREPASS")
		env = append(os.Environ(), "EMLAUNCHER_STOREPASS="+value)
	case "file":
		args = append(args, "-storepass:file", value)
	default:
		return "", errors.New(`keytool: apkfile.kspass must be "pass:..." or "file:..."`)
	}
	cmd := exec.CommandContext(ctx, keytool, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("keytool: %w: %s", err, bytes.TrimSpace(out))
	}
	return strings.TrimRight(string(out), "\n"), nil
}
