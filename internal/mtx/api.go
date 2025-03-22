package mtx

import (
	"context"
	"net/http"
	"time"

	"eaglesong.dev/gunk/internal/httpresponse"
)

type Path struct {
	Name      string    `json:"name"`
	Ready     bool      `json:"ready"`
	ReadyTime time.Time `json:"readyTime"`
	Tracks    []string  `json:"tracks"`
	Readers   []Reader  `json:"readers"`
}

type Reader struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (p *Path) Viewers() int {
	var rtsp, others int
	for _, r := range p.Readers {
		switch r.Type {
		case "hlsMuxer":
		case "rtspSession":
			rtsp += 1
		default:
			others += 1
		}
	}
	// subtract the thumbnailer and transcoder
	rtsp = max(0, rtsp-2)
	return rtsp + others
}

type API struct {
	BaseURL string
}

func (a *API) ListPaths(ctx context.Context) ([]*Path, error) {
	var body struct {
		Items []*Path `json:"items"`
	}
	if err := a.do(ctx, "/v3/paths/list", &body); err != nil {
		return nil, err
	}
	return body.Items, nil
}

func (a *API) do(ctx context.Context, path string, body any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.BaseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	return httpresponse.NewJSON(resp, body)
}
