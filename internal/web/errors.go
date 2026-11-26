package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KLab/emlauncher-go/internal/service"
)

// httpError はステータスと, 利用者に見せるメッセージを持つエラー.
// ハンドラが返すと, 画面ではエラーページ, API では JSON の {"error": msg} になる.
type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

func errorStatus(status int, msg string) error { return &httpError{status: status, msg: msg} }

func badRequest(msg string) error { return errorStatus(http.StatusBadRequest, msg) }

// エラーページのメッセージ (PHP 版と同じ文言).
var (
	errNotFound         = errorStatus(http.StatusNotFound, "Not Found")
	errPermissionDenied = errorStatus(http.StatusForbidden, "Permission Denied")
	errForbidden        = errorStatus(http.StatusForbidden, "Forbidden")
	errBadRequest       = errorStatus(http.StatusBadRequest, "Bad Request")
)

// asHTTPError はエラーをステータスとメッセージにする. 想定外のエラーは 500 (詳細は見せない).
func asHTTPError(err error) (*httpError, bool) {
	var he *httpError
	if errors.As(err, &he) {
		return he, true
	}
	var ie *service.InvalidError
	if errors.As(err, &ie) {
		return &httpError{status: http.StatusBadRequest, msg: ie.Msg}, true
	}
	if errors.Is(err, service.ErrNotFound) {
		return &httpError{status: http.StatusNotFound, msg: "Not Found"}, true
	}
	if errors.Is(err, service.ErrForbidden) {
		return &httpError{status: http.StatusForbidden, msg: "Forbidden"}, true
	}
	return &httpError{status: http.StatusInternalServerError, msg: "Internal Server Error"}, false
}

// handlerFunc はエラーを返すハンドラ. エラーの応答は page / api で包んだときに作る.
type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// page は画面のハンドラを http.HandlerFunc にする. エラーはエラーページにする.
func (s *Server) page(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			s.pageError(w, r, err)
		}
	}
}

// api は API / ajax のハンドラを http.HandlerFunc にする. エラーは JSON {"error": msg} にする.
func (s *Server) api(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			s.jsonError(w, r, err)
		}
	}
}

// pageError はエラーページを返す. 想定外のエラーは 500 (詳細は見せない).
func (s *Server) pageError(w http.ResponseWriter, r *http.Request, err error) {
	he, known := asHTTPError(err)
	if !known {
		s.log.Error("request failed", "method", r.Method, "url", r.URL.String(), "err", err)
	}
	s.renderError(w, r, he.status, he.msg)
}

// jsonError は API / ajax のエラーを JSON で返す. 想定外のエラーは PHP 版と同じく
// {"error": メッセージ, "exception": "Exception"} の 500.
func (s *Server) jsonError(w http.ResponseWriter, r *http.Request, err error) {
	he, known := asHTTPError(err)
	if !known {
		s.log.Error("request failed", "method", r.Method, "url", r.URL.String(), "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error(), "exception": "Exception"})
		return
	}
	writeJSON(w, he.status, map[string]string{"error": he.msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(b)
	return err
}
