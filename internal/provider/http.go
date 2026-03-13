package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// closeWith sets *origErr to cerr if origErr is currently nil.
func closeWith(origErr *error, c io.Closer) {
	if cerr := c.Close(); cerr != nil && *origErr == nil {
		*origErr = cerr
	}
}

// drainBody ensures the connection can be reused by reading the body.
// It ignores copy errors by design (best-effort drain).
func drainBody(b io.ReadCloser) {
	_, _ = io.Copy(io.Discard, b)
}

// doRequest runs an HTTP request, returns the response, and leaves the caller
// responsible for closing resp.Body (ideally with closeWith/defer).
func doRequest(ctx context.Context, c *http.Client, req *http.Request) (*http.Response, error) {
	resp, err := c.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// doJSON wraps a request/response cycle for JSON APIs with robust close handling.
// If status is non-2xx, the body is drained and an error is returned.
// On success, it decodes JSON into out and returns any decode error.
// All Close() calls are checked and any close error is returned if no earlier error happened.
func doJSON(ctx context.Context, c *http.Client, req *http.Request, out any) (err error) {
	resp, err := doRequest(ctx, c, req)
	if err != nil {
		return err
	}
	defer closeWith(&err, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		drainBody(resp.Body)
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	if out == nil {
		// Caller doesn't want a body; still consume to allow keep-alive reuse.
		drainBody(resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// doGET is a convenience that sets the content-type header and decodes JSON.
func doGET(ctx context.Context, c *http.Client, url string, out any) (err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	return doJSON(ctx, c, req, out)
}
