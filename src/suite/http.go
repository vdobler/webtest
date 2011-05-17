package suite

import (
	"fmt"
	"strings"
	"os"
	"http"
	"io"
	"bytes"
	"strconv"
	"mime"
)


func readBody(r io.ReadCloser) string {
	var bb bytes.Buffer
	if r != nil {
		io.Copy(&bb, r)
		r.Close()
	}
	body := bb.String()
	supertrace("Read body with len = %d:\n%s\n", len(body), body)
	return body
}

func shouldRedirect(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect:
		trace("Status code = %d: will redirect.", statusCode)
		return true
	}
	trace("Status code = %d: wont redirect.", statusCode)
	return false
}

func postWrapper(c *http.Client, t *Test) (r *http.Response, finalURL string, err os.Error) {
	return

}

func addHeadersAndCookies(req *http.Request, t *Test) {
	trace("req.Header = %v", req.Header)
	for k, v := range t.Header {
		if k == "Cookie" {
			trace("Should not send Cookies in HEADER: skipped")
		} else {
			trace("added %s = %s", k, v)
			req.Header.Set(k, v)
		}
	}

	for cn, cv := range t.Cookie {
		req.Cookie = append(req.Cookie, &http.Cookie{Name: cn, Value: cv})
	}
}

func DoAndFollow(req *http.Request) (r *http.Response, finalUrl string, cookies []*http.Cookie, err os.Error) {
	// TODO: if/when we add cookie support, the redirected request shouldn't
	// necessarily supply the same cookies as the original.
	// TODO: set referrer header on redirects.

	info("%s %s", req.Method, req.URL.String())
	r, err = http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	finalUrl = req.URL.String()
	for _, cookie := range r.SetCookie {
		trace("got cookie on first request: %s = %s", cookie.Name, cookie.Value)
		cookies = append(cookies, cookie)
	}

	if !shouldRedirect(r.StatusCode) {
		return
	}

	// Start redirecting to final destination
	r.Body.Close()
	var base = req.URL

	// Following the redirect chain is done with a clean/empty new GET request
	req = new(http.Request)
	req.Method = "GET"
	req.ProtoMajor = 1
	req.ProtoMinor = 1

	for redirect := 0; redirect < 10; redirect++ {
		var url string
		if url = r.Header.Get("Location"); url == "" {
			fmt.Printf("Header:\n%v", r.Header)
			err = os.ErrorString(fmt.Sprintf("%d response missing Location header", r.StatusCode))
			return
		}
		if base == nil {
			req.URL, err = http.ParseURL(url)
		} else {
			req.URL, err = base.ParseURL(url)
		}
		if err != nil {
			return
		}

		url = req.URL.String()
		info("GET %s", url)
		if r, err = http.DefaultClient.Do(req); err != nil {
			return
		}
		finalUrl = url
		for _, cookie := range r.SetCookie {
			// TODO check for overwriting/re-setting
			trace("got cookie on %dth request: %s = %s", redirect+1, cookie.Name, cookie.Value)
			cookies = append(cookies, cookie)
		}

		cookies = r.SetCookie

		if !shouldRedirect(r.StatusCode) {
			return
		}
		r.Body.Close()
		base = req.URL

	}
	err = os.ErrorString("stopped after 10 redirects")
	return
}

func Get(t *Test) (r *http.Response, finalUrl string, cookies []*http.Cookie, err os.Error) {
	var url = t.Url // <-- Patched

	var req http.Request
	req.Method = "GET"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
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
	req.URL, err = http.ParseURL(url)
	if err != nil {
		err = &http.URLError{"Get", url, err}
		return
	}

	addHeadersAndCookies(&req, t)
	url = req.URL.String()
	debug("Will get from %s", req.URL.String())
	r, finalUrl, cookies, err = DoAndFollow(&req)
	return
}

func hasFile(p map[string][]string) bool {
	for _, v := range p {
		if len(v) == 0 {
			continue
		}
		if strings.HasPrefix(v[0], "@file:") {
			return true
		}
	}
	return false
}

// PostForm issues a POST to the specified URL, 
// with data's keys and values urlencoded as the request body.
//
// Caller should close r.Body when done reading from it.
func Post(t *Test) (r *http.Response, finalUrl string, cookies []*http.Cookie, err os.Error) {
	var req http.Request
	var url = t.Url 
	req.Method = "POST"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Close = true
	var body *bytes.Buffer
	var contentType string
	if hasFile(t.Param) {
		body = &bytes.Buffer{}
		contentType = "multipart/form-data; "

		var boundary = fmt.Sprintf("---------------------------20718350314867") // TODO make safe
		for n, v := range t.Param {
			if len(v) > 0 && strings.HasPrefix(v[0], "@file:") {
				filename := v[0][6:]
				trace("Adding file '%s' as %s to request body.", filename, n)
				var ct string = "application/octet-stream"
				if i := strings.LastIndex(filename, "."); i != -1 {
					ct = mime.TypeByExtension(filename[i:])
					if ct == "" {
						ct = "application/octet-stream"
					}
				}
				var part = fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", boundary, n, filename)
				part += fmt.Sprintf("Content-Type: %s\r\n\r\n", ct)
				var file *os.File
				file, err = os.Open(filename)
				defer file.Close()
				if err != nil {
					warn("Cannot read from file '%s'.", filename)
					continue
				}
				body.WriteString(part)
				body.ReadFrom(file)
				body.WriteString("\r\n")
			} else {
				if len(v) > 0 {
					for _, vv := range v {
						trace("Added parameter %s with value '%s' to request body.", n, vv)
						var part = fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\n\r\n%s\r\n", boundary, n, vv) // TODO: maybe escape value?
						body.WriteString(part)
					}
				} else {
					trace("Adding empty parameter %s to request body.", n)
					var part = fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\n\r\n\r\n", boundary, n)
					body.WriteString(part)
				}
			}
		}
		body.WriteString("--" + boundary + "--\r\n") 
		contentType += "boundary=" + boundary
	} else {
		contentType = "application/x-www-form-urlencoded"
		bodystr := http.EncodeQuery(t.Param)
		body = bytes.NewBuffer([]byte(bodystr))
	}
	supertrace("Request-Body:\n%s", body.String())

	req.Body = nopCloser{body}
	req.Header = http.Header{
		"Content-Type":   {contentType},
		"Content-Length": {strconv.Itoa(body.Len())},
	}
	addHeadersAndCookies(&req, t) 

	req.ContentLength = int64(body.Len())
	req.URL, err = http.ParseURL(url)
	if err != nil {
		return nil, url, cookies, err
	}
	debug("Will post to %s", req.URL.String())

	
	dump, _ := http.DumpRequest(&req, true)
	df, err := os.Create("req.log")  // TODO filename
	if err == nil {
		df.Write(dump)
		df.Close()
	} else {
		error("Cannot open req.log: %s", err.String())
	}
	
	
	r, finalUrl, cookies, err = DoAndFollow(&req)
	return
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() os.Error { return nil }
