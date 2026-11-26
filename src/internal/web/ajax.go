package web

import (
	"errors"
	"net/http"
	"strconv"

	"emlauncher/internal/service"
)

// ajaxNotificationSetting はインストール済みのアプリの更新通知を切り替える (value: 1 / 0).
func (s *Server) ajaxNotificationSetting(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	iu, err := s.svc.SetNotify(r.Context(), f.intParam("id", 0), loginUser(r), truthy(f.Get("value")))
	if errors.Is(err, service.ErrNotFound) {
		return errorStatus(http.StatusNotFound, "installed application not found.")
	}
	if err != nil {
		return err
	}
	return writeJSON(w, http.StatusOK, map[string]any{
		"app_id":         strconv.FormatInt(iu.AppID, 10),
		"mail":           iu.Mail,
		"notify":         iu.Notify != 0,
		"last_installed": s.svc.FormatTime(iu.LastInstalled),
	})
}

type stagedJSON struct {
	FileName      string             `json:"file_name"`
	TempName      string             `json:"temp_name"`
	Platform      string             `json:"platform"`
	Identifier    *string            `json:"identifier"`
	AttachedFiles []stagedAttachJSON `json:"attached_files"`
}

type stagedAttachJSON struct {
	FileName string `json:"file_name"`
	TempName string `json:"temp_name"`
	Type     string `json:"type"`
	Size     int64  `json:"size"`
}

// ajaxUploadTemporary は画面からのアップロードの 1 段階目. ファイルを調べて一時ファイルに置き,
// その名前と調べた結果を返す (画面はそれをフォームに入れて /app/upload_post に送る).
func (s *Server) ajaxUploadTemporary(w http.ResponseWriter, r *http.Request) error {
	f := formOf(r)
	file := f.file("file")
	if file == nil {
		// PHP 版と同じ文言 ($_FILES[file] が無いときの json_encode は null)
		return badRequest("upload_file error: $_FILES[file]=null")
	}
	name := file.Name
	if name == "" {
		name = f.Get("name")
	}
	su, err := s.svc.StageUpload(r.Context(), service.LocalFile{Path: file.Path, Name: name, Size: file.Size})
	if err != nil {
		return err
	}
	res := stagedJSON{
		FileName:      su.FileName,
		TempName:      su.TempName,
		Platform:      string(su.Platform),
		AttachedFiles: []stagedAttachJSON{},
	}
	res.Identifier = su.Identifier
	for _, a := range su.Attached {
		res.AttachedFiles = append(res.AttachedFiles, stagedAttachJSON{FileName: a.FileName, TempName: a.TempName, Type: a.Type, Size: a.Size})
	}
	return writeJSON(w, http.StatusOK, res)
}
