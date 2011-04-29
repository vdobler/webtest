package suite

import (
	"fmt"
	"strings"
	"os"
	"http"
	"io"
	"bytes"
)


func readBody(r io.ReadCloser) string {
	var bb bytes.Buffer
	if r != nil {
		io.Copy(&bb, r)
		r.Close()
	}
	return bb.String()
}

func shouldRedirect(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect:
		return true
	}
	return false
}

func postWrapper(c *http.Client, t *Test) (r *http.Response, finalURL string, err os.Error) {
	return

}

func addHeaders(req *http.Request, t *Test) {
	for k, v := range t.Header {
		trace("req.Header = %v", req.Header)
		req.Header.Set(k, v)
	}
}


func Get(t *Test) (r *http.Response, finalURL string, err os.Error) {
	var url = t.Url // <-- Patched
	// TODO: if/when we add cookie support, the redirected request shouldn't
	// necessarily supply the same cookies as the original.
	// TODO: set referrer header on redirects.
	var base *http.URL
	// TODO: remove this hard-coded 10 and use the Client's policy
	// (ClientConfig) instead.
	for redirect := 0; ; redirect++ {
		if redirect >= 10 {
			err = os.ErrorString("stopped after 10 redirects")
			break
		}

		var req http.Request
		req.Method = "GET"
		req.ProtoMajor = 1
		req.ProtoMinor = 1
		// vvvv Patched vvvv
		req.Header = http.Header{}
		if len(t.Param) > 0 {
			ep := http.EncodeQuery(t.Param)
			// TODO handle #-case
			if strings.Contains(url, "?") {
				url = url + "&" + ep
			} else {
				url = url + "?" + ep
			}
		}
		// ^^^^ Patched ^^^^
		if base == nil {
			req.URL, err = http.ParseURL(url)
		} else {
			req.URL, err = base.ParseURL(url)
		}
		if err != nil {
			break
		}
		// vvvv Patched vvvv
		addHeaders(&req, t)
		// ^^^^ Patched ^^^^
		url = req.URL.String()
		trace("GETing %s", url)
		if r, err = http.DefaultClient.Do(&req); err != nil {
			break
		}
		if shouldRedirect(r.StatusCode) {
			r.Body.Close()
			if url = r.Header.Get("Location"); url == "" {
				err = os.ErrorString(fmt.Sprintf("%d response missing Location header", r.StatusCode))
				break
			}
			base = req.URL
			continue
		}
		finalURL = url
		return
	}

	err = &http.URLError{"Get", url, err}
	return
}
