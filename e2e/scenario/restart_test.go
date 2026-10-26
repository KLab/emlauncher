package scenario

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	e2e "emlauncher-e2e"
)

// TestGracefulRestart は無停止の再起動 (SIGHUP) を確認する. Go 版のみ.
//   - 再起動の間も, 新しいリクエストが失敗しない
//   - 再起動前に始まっていたアップロードは, 古いプロセスが最後まで処理する
//   - 古いプロセスは処理を終えたら終了する
func TestGracefulRestart(t *testing.T) {
	e := e2e.New(t)
	if e.Config.Target == "php" {
		t.Skip("無停止の再起動は Go 版のみ")
	}
	owner := e.NewUser("owner")
	app := e.CreateApp(owner, e2e.Unique("app"))
	oldPID := e.ServerPID()

	// 1. アップロードを途中まで送る
	apk := e2e.APK("com.example.restart")
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	req, err := http.NewRequest(http.MethodPost, e.Config.BaseURL+"/api/upload", pr)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	type result struct {
		res *http.Response
		err error
	}
	uploaded := make(chan result, 1)
	go func() {
		res, err := http.DefaultClient.Do(req)
		uploaded <- result{res, err}
	}()
	mw.WriteField("api_key", app.APIKey)
	mw.WriteField("title", "uploaded across restart")
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="restart.apk"`)
	h.Set("Content-Type", "application/vnd.android.package-archive")
	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(apk[:len(apk)/2])
	time.Sleep(500 * time.Millisecond) // サーバがリクエストを受け付けて本文を読み始めるまで

	// 2. 再起動の間, 別のリクエストを送り続ける
	stop := make(chan struct{})
	var polls, failures atomic.Int64
	var failMsg sync.Map
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c := &http.Client{Timeout: 10 * time.Second}
		for {
			select {
			case <-stop:
				return
			default:
			}
			res, err := c.Get(e.Config.BaseURL + "/health")
			polls.Add(1)
			if err != nil {
				failures.Add(1)
				failMsg.Store(err.Error(), true)
			} else {
				io.Copy(io.Discard, res.Body)
				res.Body.Close()
				if res.StatusCode != http.StatusOK {
					failures.Add(1)
					failMsg.Store(fmt.Sprint("status ", res.StatusCode), true)
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	e.ReloadServer()
	newPID := waitFor(t, "new process", func() (string, bool) {
		pid := e.ServerPID()
		return pid, pid != oldPID
	})
	t.Logf("pid %s -> %s", oldPID, newPID)
	// systemd で動かしている場合, 古いプロセスが処理を終えるまではメインプロセスは古いまま.
	// 古いプロセスが終了した時点で PID ファイルから新しいプロセスに移る (下で確認する)
	if pid := e.MainPID(); e.Config.GoMainPID != "" && pid != oldPID && pid != newPID {
		t.Fatalf("systemd MainPID = %s, want %s or %s", pid, oldPID, newPID)
	}

	// 新しいプロセスがリクエストを処理する (API も DB も使える)
	res := e.APICall("package_list", app.APIKey, nil)
	e2e.ExpectStatus(t, res, http.StatusOK)
	if !e.ProcessAlive(oldPID) {
		t.Fatal("old process exited before the in-flight upload finished")
	}

	// 3. アップロードの残りを送る. 古いプロセスが最後まで処理する
	part.Write(apk[len(apk)/2:])
	mw.Close()
	pw.Close()
	r := <-uploaded
	if r.err != nil {
		t.Fatalf("upload across restart: %v", r.err)
	}
	body, _ := io.ReadAll(r.res.Body)
	r.res.Body.Close()
	if r.res.StatusCode != http.StatusOK {
		t.Fatalf("upload across restart: status %d: %s", r.res.StatusCode, body)
	}
	pkg := e.DB.Row("SELECT * FROM package WHERE app_id = ? AND title = 'uploaded across restart'", app.ID)
	checkStored(t, e, e2e.PackageKey(pkg), apk, mimeStored, false)

	// 4. 古いプロセスは終了する. 新しいプロセスは動き続ける (systemd がサービスごと止めたりしない)
	waitFor(t, "old process to exit", func() (string, bool) {
		return "", !e.ProcessAlive(oldPID)
	})
	time.Sleep(time.Second)
	if !e.ProcessAlive(newPID) {
		t.Fatal("new process exited after the old process exited")
	}
	if e.Config.GoMainPID != "" {
		waitFor(t, "systemd MainPID to be the new process", func() (string, bool) {
			pid := e.MainPID()
			return pid, pid == newPID
		})
		// 再起動 (Restart=always) されずに動き続けている
		if out, _ := e.GoExec("systemctl show -p ActiveState --value emlauncher"); out != "active" {
			t.Fatalf("ActiveState after restart = %q", out)
		}
		if out, _ := e.GoExec("systemctl show -p NRestarts --value emlauncher"); out != "0" {
			t.Fatalf("NRestarts after restart = %q", out)
		}
	}
	e2e.ExpectStatus(t, e.NewClient().Get("/health"), http.StatusOK)

	close(stop)
	wg.Wait()
	if failures.Load() > 0 {
		var msgs []string
		failMsg.Range(func(k, _ any) bool { msgs = append(msgs, k.(string)); return true })
		t.Fatalf("%d / %d requests failed during restart: %v", failures.Load(), polls.Load(), msgs)
	}
	if polls.Load() < 10 {
		t.Fatalf("too few requests during restart: %d", polls.Load())
	}
	t.Logf("%d requests during restart, no failure", polls.Load())
}

// TestGracefulRestartFailure は, 新しいプロセスが起動できない (設定の誤り) とき,
// 古いプロセスが動き続けることを確認する. Go 版のみ.
func TestGracefulRestartFailure(t *testing.T) {
	e := e2e.New(t)
	if e.Config.Target == "php" {
		t.Skip("無停止の再起動は Go 版のみ")
	}
	orig, err := os.ReadFile(e.Config.GoConfig)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	t.Cleanup(func() { os.WriteFile(e.Config.GoConfig, orig, 0o644) })
	if err := os.WriteFile(e.Config.GoConfig, append(append([]byte{}, orig...), []byte("\nno_such_key: 1\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	oldPID := e.ServerPID()
	e.ReloadServer()
	time.Sleep(3 * time.Second)
	if pid := e.ServerPID(); pid != oldPID {
		t.Fatalf("process replaced with a broken config: %s -> %s", oldPID, pid)
	}
	if !e.ProcessAlive(oldPID) {
		t.Fatal("old process exited")
	}
	e2e.ExpectStatus(t, e.NewClient().Get("/health"), http.StatusOK)
	if pid := e.MainPID(); e.Config.GoMainPID != "" && pid != oldPID {
		t.Fatalf("systemd MainPID = %s, want %s", pid, oldPID)
	}

	// 設定を直せば再起動できる
	if err := os.WriteFile(e.Config.GoConfig, orig, 0o644); err != nil {
		t.Fatal(err)
	}
	e.ReloadServer()
	waitFor(t, "new process", func() (string, bool) {
		pid := e.ServerPID()
		return pid, pid != oldPID
	})
	e2e.ExpectStatus(t, e.NewClient().Get("/health"), http.StatusOK)
}

// waitFor は cond が true になるまで (最大 30 秒) 待つ.
func waitFor(t *testing.T, what string, cond func() (string, bool)) string {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		v, ok := cond()
		if ok {
			return v
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %s", what)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// TestSystemdStopStart は systemd で動かしているとき, 無停止の再起動でプロセスが入れ替わった後でも
// systemd がプロセスを把握していること (異常終了したら再起動する, systemctl stop / start が効く) を確認する.
func TestSystemdStopStart(t *testing.T) {
	e := e2e.New(t)
	if e.Config.GoMainPID == "" {
		t.Skip("systemd で動かしているときのみ")
	}
	// 入れ替わった後のプロセスに対して行う
	oldPID := e.ServerPID()
	e.ReloadServer()
	newPID := waitFor(t, "new process", func() (string, bool) {
		pid := e.ServerPID()
		return pid, pid != oldPID
	})
	waitFor(t, "old process to exit", func() (string, bool) { return "", !e.ProcessAlive(oldPID) })
	waitFor(t, "systemd MainPID to be the new process", func() (string, bool) {
		pid := e.MainPID()
		return pid, pid == newPID
	})

	// 異常終了したら systemd が再起動する (Restart=always)
	restarts := func() string {
		out, err := e.GoExec("systemctl show -p NRestarts --value emlauncher")
		if err != nil {
			t.Fatalf("NRestarts: %v: %s", err, out)
		}
		return out
	}
	before := restarts()
	if out, err := e.GoExec("kill -9 " + newPID); err != nil {
		t.Fatalf("kill: %v: %s", err, out)
	}
	newPID = waitFor(t, "systemd to restart the service", func() (string, bool) {
		pid := e.MainPID()
		return pid, restarts() != before && pid != "0" && pid != newPID && e.ProcessAlive(pid)
	})
	waitFor(t, "service to be ready", func() (string, bool) {
		res, err := http.Get(e.Config.BaseURL + "/health")
		if err != nil {
			return "", false
		}
		res.Body.Close()
		return "", res.StatusCode == http.StatusOK
	})

	if out, err := e.GoExec("systemctl stop emlauncher"); err != nil {
		t.Fatalf("stop: %v: %s", err, out)
	}
	if out, _ := e.GoExec("systemctl is-active emlauncher"); out != "inactive" {
		t.Fatalf("is-active after stop = %q", out)
	}
	if e.ProcessAlive(newPID) {
		t.Fatalf("process %s is still running after stop", newPID)
	}
	if pids := e.ServerProcesses(); len(pids) > 0 {
		t.Fatalf("emlauncher processes remain after stop: %v", pids)
	}

	if out, err := e.GoExec("systemctl start emlauncher"); err != nil {
		t.Fatalf("start: %v: %s", err, out)
	}
	waitFor(t, "service to be ready", func() (string, bool) {
		res, err := http.Get(e.Config.BaseURL + "/health")
		if err != nil {
			return "", false
		}
		res.Body.Close()
		return "", res.StatusCode == http.StatusOK
	})
	if pid, main := e.ServerPID(), e.MainPID(); pid != main {
		t.Fatalf("pid file %s != MainPID %s", pid, main)
	}
}
