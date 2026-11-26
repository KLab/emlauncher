package pkgfile

import (
	"archive/zip"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"howett.net/plist"
)

// testdata/ のファイルは e2e の go run ./cmd/genfixture で作ったもの.

func TestExt(t *testing.T) {
	for in, want := range map[string]string{
		"a.apk": "apk", "a.dSYM.zip": "zip", "noext": "", "dir.x/file": "", "a.": "", `C:\x\y.ipa`: "ipa",
	} {
		if got := Ext(in); got != want {
			t.Errorf("Ext(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetect(t *testing.T) {
	text := filepath.Join(t.TempDir(), "readme.txt")
	if err := os.WriteFile(text, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path, name string
		want       Kind
	}{
		{"testdata/sample.ipa", "sample.ipa", KindIPA},
		{"testdata/sample.apk", "my app.apk", KindAPK},
		{"testdata/sample.aab", "b.aab", KindAAB},
		{"testdata/sample.apk", "sample.zip", KindUnknown},
		{"testdata/sample.apk", "SAMPLE.APK", KindUnknown}, // PHP 版と同じく拡張子は大文字小文字を区別する
		{text, "fake.apk", KindUnknown},
		{text, "readme.txt", KindUnknown},
	} {
		got, err := Detect(tc.path, tc.name)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("Detect(%s, %s) = %v, want %v", tc.path, tc.name, got, tc.want)
		}
	}
	if KindAAB.Platform() != Android || KindIPA.Platform() != IOS || KindUnknown.Platform() != Unknown {
		t.Error("Platform")
	}
}

func TestIPABundleID(t *testing.T) {
	id, err := IPABundleID("testdata/sample.ipa")
	if err != nil || id != "com.example.e2e.ios" {
		t.Fatalf("IPABundleID = %q, %v", id, err)
	}

	// 実際の IPA はバイナリ形式の plist が多い
	data, err := plist.Marshal(map[string]any{"CFBundleIdentifier": "com.example.binary", "CFBundleVersion": "1"}, plist.BinaryFormat)
	if err != nil {
		t.Fatal(err)
	}
	path := writeZip(t, map[string][]byte{"Payload/App.app/Info.plist": data, "Payload/App.app/Frameworks/X.framework/Info.plist": []byte("x")})
	if id, err := IPABundleID(path); err != nil || id != "com.example.binary" {
		t.Fatalf("binary plist: %q, %v", id, err)
	}

	if _, err := IPABundleID(writeZip(t, map[string][]byte{"Payload/App.app/App": []byte("x")})); err == nil {
		t.Fatal("ipa without Info.plist accepted")
	}

	// CFBundleIdentifier が無ければ空 (identifier は NULL になる)
	data, err = plist.Marshal(map[string]any{"CFBundleVersion": "1"}, plist.XMLFormat)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := IPABundleID(writeZip(t, map[string][]byte{"Payload/App.app/Info.plist": data})); err != nil || id != "" {
		t.Fatalf("without CFBundleIdentifier: %q, %v", id, err)
	}
}

func TestAPKPackageName(t *testing.T) {
	name, err := APKPackageName("testdata/sample.apk")
	if err != nil || name != "com.example.e2e.android" {
		t.Fatalf("APKPackageName = %q, %v", name, err)
	}
	if _, err := APKPackageName(writeZip(t, map[string][]byte{"AndroidManifest.xml": []byte("garbage")})); err == nil {
		t.Fatal("broken manifest accepted")
	}
	if _, err := APKPackageName("testdata/sample.ipa"); err == nil {
		t.Fatal("apk without manifest accepted")
	}
}

// bundletool の実行は Java と bundletool がある環境 (EML_TEST_BUNDLETOOL に jar のパス) でだけ確かめる.
func TestUniversalAPK(t *testing.T) {
	jar := os.Getenv("EML_TEST_BUNDLETOOL")
	if jar == "" {
		t.Skip("set EML_TEST_BUNDLETOOL to run")
	}
	dir := t.TempDir()
	ks := filepath.Join(dir, "test.keystore")
	out, err := exec.Command("keytool", "-genkeypair", "-keystore", ks, "-alias", "test", "-storepass", "testpass",
		"-keypass", "testpass", "-keyalg", "RSA", "-keysize", "2048", "-validity", "1", "-dname", "CN=test").CombinedOutput()
	if err != nil {
		t.Fatalf("keytool: %v: %s", err, out)
	}
	b := &Bundletool{Java: []string{"java"}, Jar: jar, Keystore: ks, KSPass: "pass:testpass", KeyAlias: "test", KeyPass: "pass:testpass"}
	apk := filepath.Join(dir, "universal.apk")
	if err := b.UniversalAPK(context.Background(), "testdata/sample.aab", apk); err != nil {
		t.Fatal(err)
	}
	if name, err := APKPackageName(apk); err != nil || name != "com.example.e2e.bundle" {
		t.Fatalf("APKPackageName = %q, %v", name, err)
	}
	info, err := b.KeystoreInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Log(info)
}

func writeZip(t *testing.T, files map[string][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}
