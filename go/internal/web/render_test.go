package web

import "testing"

func TestLoadTemplates(t *testing.T) {
	ts, err := loadTemplates()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"top/index", "app/index", "app/new", "app/comment", "app/preferences", "app/upload",
		"package/index", "package/edit", "package/delete_confirm", "package/attach_delete_confirm",
		"package/create_token", "package/guestpass", "myapps/installed", "myapps/own", "guestpass/list",
		"login/index", "login/password_reset", "doc/api", "doc/api_upload", "doc/keystore", "_error"} {
		if ts[name] == nil {
			t.Errorf("template %s is not loaded", name)
		}
	}
}

func TestPaging(t *testing.T) {
	nums := func(v pagingView) (out []int) {
		for _, it := range v.Items {
			if it.Omission {
				out = append(out, 0)
			} else {
				out = append(out, it.Num)
			}
		}
		return
	}
	for _, tc := range []struct {
		cur, max int
		want     []int
	}{
		{1, 3, []int{1, 2, 3}},
		{1, 10, []int{1, 2, 3, 4, 5, 6, 0, 10}},
		{10, 10, []int{1, 0, 5, 6, 7, 8, 9, 10}},
		{5, 10, []int{1, 0, 3, 4, 5, 6, 7, 0, 10}},
		{1, 0, nil},
	} {
		got := nums(paging("http://x/?id=1", tc.cur, tc.max))
		if len(got) != len(tc.want) {
			t.Errorf("paging(%d, %d) = %v, want %v", tc.cur, tc.max, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("paging(%d, %d) = %v, want %v", tc.cur, tc.max, got, tc.want)
				break
			}
		}
	}
}

func TestFormatSize(t *testing.T) {
	for n, want := range map[int64]string{
		500: "500 B", 1024: "1 KB", 1536: "1.5 KB", 10 * 1024 * 1024: "10 MB", 1288490189: "1.2 GB",
	} {
		if got := formatSize(n); got != want {
			t.Errorf("formatSize(%d) = %q, want %q", n, got, want)
		}
	}
}
