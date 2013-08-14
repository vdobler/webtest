package suite

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

func init() {
	mime.AddExtensionType(".html", "text/html")
	// TODO: others...
}

func escapeQuotes(s string) string {
	s = strings.Replace(s, "\\", "\\\\", -1)
	s = strings.Replace(s, "\"", "\\\"", -1)
	return s

}

func readBody(r io.ReadCloser) []byte {
	var bb bytes.Buffer
	if r != nil {
		io.Copy(&bb, r)
		r.Close()
	}
	tracef("Read body with len = %d.", bb.Len())
	return bb.Bytes()
}

// Determine wether statusCode tells us to redirect
func shouldRedirect(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect:
		tracef("Status code = %d: will redirect.", statusCode)
		return true
	}
	tracef("Status code = %d: wont redirect.", statusCode)
	return false
}

func postWrapper(c *http.Client, t *Test) (r *http.Response, finalURL string, err error) {
	return

}

// Add header fields and cookies from test t to request req.
func addHeadersAndCookies(req *http.Request, t *Test) {
	for k, v := range t.Header {
		if k == "Cookie" {
			tracef("Should not send Cookies in HEADER: skipped")
		} else {
			tracef("added %s = %s", k, v)
			req.Header.Set(k, v)
		}
	}

	for _, cookie := range t.Jar.Select(req.URL) {
		tracef("Will send cookie %s=%s", cookie.Name, cookie.Value)
		req.AddCookie(cookie)
	}
}

// Dump request req in wire format to dump if non nil.
func dumpReq(req *http.Request, dump io.Writer) {
	if dump != nil {
		rd, err := httputil.DumpRequest(req, true)
		if err == nil {
			dump.Write(rd)
			dump.Write([]byte("\r\n\r\n--------------------------------------------------------------------------------------\r\n"))
			dump.Write([]byte("--------------------------------------------------------------------------------------\r\n\r\n\r\n"))
		} else {
			errorf("Cannot dump request: %s", err.Error())
		}
	}
}

// Dump response in wire format to dump if non nil.
func dumpRes(res *http.Response, dump io.Writer) {
	if dump != nil {
		rd, err := httputil.DumpResponse(res, true)
		if err == nil {
			dump.Write(rd)
			dump.Write([]byte("\r\n\r\n======================================================================================\r\n"))
			dump.Write([]byte("======================================================================================\r\n\r\n\r\n"))
		} else {
			errorf("Cannot dump response: %s", err.Error())
		}
	}
}

/*
 Cookie Handling
 ---------------

 First Problem
  - GET domain.net/ with cookie a=vala
  - redirect to other.org/
  - GET other.org/  Send cookie?  Depends on domain of cookie
 Solution:
 A very simple cookiejar.

 Second Problem
  - GET domain.net/a with cookie a=vala
  - redirect to domain.net/b with "Set-Cookie: a=; Max-Age: 0" aka delete cookie
  - GET domain.net/b  Send Cookie?  What to report?
 Solution:
 a very simple Cookie jar.

 Header Values
 -------------
 Problem: Which Headers to send on a redirect chain?
 Solution: All the ones requested...

*/

//
func valid(cookie *http.Cookie) bool {
	if cookie.MaxAge < 0 {
		tracef("Cookie %s has MaxAge < 0.", cookie.Name)
		return false
	}

	if cookie.Expires.Year() != 0 {
		if cookie.Expires.Before(time.Now()) {
			tracef("Cookie %s has expired.", cookie.Name)
			return false
		}
	}

	tracef("Cookie %s valid: MaxAge = %d, Expires = %s", cookie.Name, cookie.MaxAge, cookie.Expires.Format(http.TimeFormat))
	return true
}

// A client which does not follow any redirects.
var nonfollowingClient http.Client = http.Client{
	Transport: nil,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 {
			return errors.New("WE DONT FOLLOW")
		}
		return nil
	},
}

func redirectChecker(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return nil
}

// Perform the request and follow up to 10 redirects.
// All cookie setting are collected, the final URL is reported.
func DoAndFollow(ireq *http.Request, t *Test) (r *http.Response, finalUrl string, cookies []*http.Cookie, err error) {
	infof("%s %s", ireq.Method, ireq.URL.String())

	var base *url.URL

	var via []*http.Request

	req := ireq
	addHeadersAndCookies(req, t)

	urlStr := "" // next relative or absolute URL to fetch (after first request)
	for redirect := 0; ; redirect++ {
		if redirect != 0 {
			req = new(http.Request)
			req.Method = ireq.Method
			req.Header = make(http.Header)
			req.URL, err = base.Parse(urlStr)
			if err != nil {
				break
			}
			addHeadersAndCookies(req, t)
			if len(via) > 0 {
				// Add the Referer header.
				lastReq := via[len(via)-1]
				if lastReq.URL.Scheme != "https" {
					req.Header.Set("Referer", lastReq.URL.String())
				}

				err = redirectChecker(req, via)
				if err != nil {
					break
				}
			}

		}
		dumpReq(req, t.Dump)
		urlStr = req.URL.String()
		if r, err = nonfollowingClient.Do(req); err != nil {
			if strings.HasSuffix(err.Error(), "WE DONT FOLLOW") {
				err = nil
			} else {
				return
			}
		}
		dumpRes(r, t.Dump)

		finalUrl = r.Request.URL.String()
		cd := stripPort(req.URL.Host)
		for _, c := range r.Cookies() {
			if c.Domain == "" {
				c.Domain = cd
			}
			t.Jar.Update(*c, req.URL.Host)
			cookies = append(cookies, c)
		}

		if shouldRedirect(r.StatusCode) {
			if r.Body != nil {
				r.Body.Close()
			}
			if urlStr = r.Header.Get("Location"); urlStr == "" {
				err = fmt.Errorf("%d response missing Location header", r.StatusCode)
				break
			}
			base = req.URL
			via = append(via, req)
			continue
		}
		return
	}

	method := ireq.Method
	err = &url.Error{method[0:1] + strings.ToLower(method[1:]), urlStr, err}
	return

}

// Perform a GET request for the test t.
func Get(t *Test) (r *http.Response, finalUrl string, cookies []*http.Cookie, err error) {
	var testurl = t.Url // <-- Patched

	if len(t.Param) > 0 {
		values := make(url.Values)
		for k, vs := range t.Param {
			for _, v := range vs {
				values.Add(k, v)
			}
		}

		ep := values.Encode()
		if strings.Contains(testurl, "?") {
			testurl = testurl + "&" + ep
		} else {
			testurl = testurl + "?" + ep
		}
	}

	req, err := http.NewRequest("GET", testurl, nil)
	if err != nil {
		return
	}

	debugf("Will get from %s", req.URL.String())
	r, finalUrl, cookies, err = DoAndFollow(req, t)
	return
}

// Return true if the parameters contain a file
func hasFile(param *map[string][]string) bool {
	for _, v := range *param {
		if len(v) == 0 {
			continue
		}
		if strings.HasPrefix(v[0], "@file:") {
			tracef("File to upload present.")
			return true
		}
	}
	return false
}

// Format the parameter map as a multipart message body.
func multipartBody(param *map[string][]string) (*bytes.Buffer, string) {
	var body *bytes.Buffer = &bytes.Buffer{}

	var mpwriter = multipart.NewWriter(body)
	// All non-file parameters come first
	for n, v := range *param {
		if len(v) > 0 && strings.HasPrefix(v[0], "@file:") {
			continue // files go at the end
		}
		if len(v) > 0 {
			for _, vv := range v {
				tracef("Added parameter %s with value '%s' to request body.", n, vv)
				mpwriter.WriteField(n, vv)
			}
		} else {
			tracef("Adding empty parameter %s to request body.", n)
			mpwriter.WriteField(n, "")
		}
	}

	// File parameters at bottom
	for n, v := range *param {
		if !(len(v) > 0 && strings.HasPrefix(v[0], "@file:")) {
			continue // allready written
		}
		filename := v[0][6:]
		tracef("Adding file '%s' as %s to request body.", filename, n)
		var ct string = "application/octet-stream"
		if i := strings.LastIndex(filename, "."); i != -1 {
			ct = mime.TypeByExtension(filename[i:])
			if ct == "" {
				ct = "application/octet-stream"
			}
		}

		basename := path.Base(filename)

		// Fix until CreateFormFile honours variable contentTypes
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
				escapeQuotes(n), escapeQuotes(basename)))
		h.Set("Content-Type", ct)
		fw, err := mpwriter.CreatePart(h)
		// fw, err := mpwriter.CreateFormFile(n, basename)

		if err != nil {
			warnf("Cannot write file multipart: ", err.Error())
			continue
		}

		file, err := os.Open(filename)
		defer file.Close()
		if err != nil {
			warnf("Cannot read from file '%s': %s.", filename, err.Error())
			continue
		}
		io.Copy(fw, file)
	}
	mpwriter.Close()

	return body, mpwriter.Boundary()
}

// PostForm issues a POST to the specified URL,
// with data's keys and values urlencoded as the request body.
//
// Caller should close r.Body when done reading from it.
func Post(t *Test) (r *http.Response, finalUrl string, cookies []*http.Cookie, err error) {
	var body *bytes.Buffer
	var contentType string
	if hasFile(&t.Param) || t.Method == "POST:mp" {
		var boundary string
		body, boundary = multipartBody(&t.Param)
		contentType = "multipart/form-data; boundary=" + boundary
	} else {
		contentType = "application/x-www-form-urlencoded"
		values := make(url.Values)
		for k, vs := range t.Param {
			for _, v := range vs {
				values.Add(k, v)
			}
		}
		bodystr := values.Encode()
		body = bytes.NewBuffer([]byte(bodystr))
	}

	req, err := http.NewRequest("POST", t.Url, body)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", contentType)
	addHeadersAndCookies(req, t)

	debugf("Will post to %s", req.URL.String())

	r, finalUrl, cookies, err = DoAndFollow(req, t)
	return
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }
