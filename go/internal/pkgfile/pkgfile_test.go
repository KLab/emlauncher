package pkgfile

import (
	"archive/zip"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"howett.net/plist"

	"emlauncher/internal/config"
)

// testdata は e2e の genfixture で生成したもの (universal.apk は bundletool で sample.aab から生成).

func TestExt(t *testing.T) {
	for name, want := range map[string]string{
		"a.apk": "apk", "a.b.ipa": "ipa", "noext": "", "dir.x/file": "", "x.tar.gz": "gz", "C:\\tmp\\a.aab": "aab",
	} {
		if got := Ext(name); got != want {
			t.Errorf("Ext(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestDetect(t *testing.T) {
	for _, tc := range []struct {
		name, file, platform string
	}{
		{"sample.ipa", "testdata/sample.ipa", PlatformIOS},
		{"sample.apk", "testdata/sample.apk", PlatformAndroid},
		{"sample.aab", "testdata/sample.aab", PlatformAndroid},
		{"sample.zip", "testdata/sample.apk", PlatformUnknown}, // 拡張子が違う
		{"fake.apk", "testdata/icon.png", PlatformUnknown},     // zip ではない
	} {
		pf, _, err := Detect(tc.name, tc.file)
		if err != nil || pf != tc.platform {
			t.Errorf("Detect(%s, %s) = %q, %v; want %q", tc.name, tc.file, pf, err, tc.platform)
		}
	}
}

func TestIOSBundleID(t *testing.T) {
	id, err := IOSBundleID("testdata/sample.ipa")
	if err != nil || id != "com.example.e2e.ios" {
		t.Errorf("got %q, %v", id, err)
	}

	// バイナリ形式の Info.plist
	b, err := plist.Marshal(map[string]any{"CFBundleIdentifier": "com.example.binary"}, plist.BinaryFormat)
	if err != nil {
		t.Fatal(err)
	}
	p := writeZip(t, map[string][]byte{"Payload/Bin.app/Info.plist": b})
	if id, err := IOSBundleID(p); err != nil || id != "com.example.binary" {
		t.Errorf("binary plist: %q, %v", id, err)
	}

	// Info.plist が無い / 入れ子のアプリの Info.plist は対象外
	p = writeZip(t, map[string][]byte{"Payload/A.app/Frameworks/X.framework/Info.plist": b})
	if _, err := IOSBundleID(p); err == nil {
		t.Error("no error without Payload/*.app/Info.plist")
	}
}

func TestAndroidPackageName(t *testing.T) {
	for file, want := range map[string]string{
		"testdata/sample.apk":    "com.example.e2e.android",
		"testdata/universal.apk": "com.example.e2e.bundle",
	} {
		if got, err := AndroidPackageName(file); err != nil || got != want {
			t.Errorf("%s: %q, %v; want %q", file, got, err, want)
		}
	}
	if _, err := AndroidPackageName("testdata/sample.ipa"); err == nil {
		t.Error("no error for ipa")
	}
}

// bundletool と java がある環境 (EML_TEST_BUNDLETOOL=/path/to/bundletool.jar) でのみ実行する.
func TestUniversalAPK(t *testing.T) {
	jar := os.Getenv("EML_TEST_BUNDLETOOL")
	if jar == "" {
		t.Skip("EML_TEST_BUNDLETOOL is not set")
	}
	if _, err := exec.LookPath("keytool"); err != nil {
		t.Skip("keytool not found")
	}
	ks := filepath.Join(t.TempDir(), "test.keystore")
	out, err := exec.Command("keytool", "-genkeypair", "-keystore", ks, "-alias", "test", "-storepass", "testpass",
		"-keypass", "testpass", "-keyalg", "RSA", "-keysize", "2048", "-validity", "1", "-dname", "CN=test").CombinedOutput()
	if err != nil {
		t.Fatalf("keytool: %v %s", err, out)
	}
	b := NewBundletool(config.APKFile{Java: "java -Xmx512m", Bundletool: jar, Keystore: ks,
		KSPass: "pass:testpass", KeyAlias: "test", KeyPass: "pass:testpass"})
	apk, err := b.UniversalAPK(context.Background(), "testdata/sample.aab")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(apk)
	if got, err := AndroidPackageName(apk); err != nil || got != "com.example.e2e.bundle" {
		t.Errorf("package = %q, %v", got, err)
	}
	info, err := b.KeystoreInfo(context.Background())
	if err != nil || len(info) == 0 {
		t.Errorf("KeystoreInfo: %v", err)
	}
}

func writeZip(t *testing.T, files map[string][]byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "x.zip")
	f, _ := os.Create(p)
	zw := zip.NewWriter(f)
	for name, data := range files {
		w, _ := zw.Create(name)
		w.Write(data)
	}
	zw.Close()
	f.Close()
	return p
}
