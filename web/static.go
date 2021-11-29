package web

import (
	"errors"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/hlog"
)

type staticServer struct {
	Root string
}

func (s staticServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fp := filepath.Join(s.Root, filepath.FromSlash(req.URL.Path))
	if fp == s.Root {
		fp = filepath.Join(s.Root, "index.html")
	} else if !strings.HasPrefix(fp, s.Root) {
		http.Error(rw, "", http.StatusForbidden)
		return
	}
	var f *os.File
	var err error
	brotli, gzip := checkEncodings(req.Header.Get("Accept-Encoding"))
	if f == nil && brotli {
		f, err = os.Open(fp + ".br")
		if err == nil {
			rw.Header().Set("Content-Encoding", "br")
		} else if !errors.Is(err, os.ErrNotExist) {
			hlog.FromRequest(req).Err(err).Msg("failed serving UI")
			http.Error(rw, "", http.StatusForbidden)
			return
		}
	}
	if f == nil && gzip {
		f, err = os.Open(fp + ".gz")
		if err == nil {
			rw.Header().Set("Content-Encoding", "gzip")
		} else if !errors.Is(err, os.ErrNotExist) {
			hlog.FromRequest(req).Err(err).Msg("failed serving UI")
			http.Error(rw, "", http.StatusForbidden)
			return
		}
	}
	if f == nil {
		f, err = os.Open(fp)
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(rw, req)
			return
		} else if err != nil {
			hlog.FromRequest(req).Err(err).Msg("failed serving UI")
			http.Error(rw, "", http.StatusForbidden)
			return
		}
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		hlog.FromRequest(req).Err(err).Msg("failed serving UI")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	} else if st.IsDir() {
		http.Error(rw, "", http.StatusForbidden)
		return
	}
	ctype := mime.TypeByExtension(filepath.Ext(fp))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	rw.Header().Set("Content-Type", ctype)
	http.ServeContent(rw, req, "", st.ModTime(), f)
}

func checkEncodings(ae string) (brotli, gzip bool) {
	for {
		i := strings.IndexByte(ae, ';')
		j := strings.IndexByte(ae, ',')
		switch {
		case i < 0 && j < 0:
			i = len(ae)
		case j < 0:
		case i < 0, j < i:
			i = j
		}
		switch strings.TrimSpace(ae[:i]) {
		case "br":
			brotli = true
		case "gzip":
			gzip = true
		}
		if j < 0 {
			break
		}
		ae = ae[j+1:]
	}
	return
}
