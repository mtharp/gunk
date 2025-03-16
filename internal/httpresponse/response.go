package httpresponse

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	maxError = 10e3
	maxBody  = 10e6
)

type Error struct {
	Response *http.Response
	Body     []byte
}

func (e Error) Error() string {
	return fmt.Sprintf("HTTP %s during %s %s:\n%s", e.Response.Status, e.Response.Request.Method, e.Response.Request.URL, string(e.Body))
}

func NewError(resp *http.Response) Error {
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxError))
	return Error{
		Response: resp,
		Body:     body,
	}
}

func NewJSON(resp *http.Response, dest any) error {
	if resp.StatusCode != http.StatusOK {
		return NewError(resp)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "json") {
		return fmt.Errorf("%s %s: unexpected Content-Type %q", resp.Request.Method, resp.Request.URL, ct)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return fmt.Errorf("%s %s: parsing JSON body: %w", resp.Request.Method, resp.Request.URL, err)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("%s %s: parsing JSON body: %w", resp.Request.Method, resp.Request.URL, err)
	}
	return nil
}
