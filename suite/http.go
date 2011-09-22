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
	"mime/multipart"
	"rand"
	"time"
	"path"
	"url"
)

func readBody(r io.ReadCloser) []byte {
	var bb bytes.Buffer
	if r != nil {
		io.Copy(&bb, r)
		r.Close()
	}
	trace("Read body with len = %d.", bb.Len())
	return bb.Bytes()
}

// Determine wether statusCode tells us to redirect
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

// Add header fields and cookies from test t to request req.
func addHeadersAndCookies(req *http.Request, t *Test) {
	for k, v := range t.Header {
		if k == "Cookie" {
			trace("Should not send Cookies in HEADER: skipped")
		} else {
			trace("added %s = %s", k, v)
			req.Header.Set(k, v)
		}
	}

	for cn, cv := range t.Cookie {
		req.AddCookie(&http.Cookie{Name: cn, Value: cv})
	}
}

// Dump request req in wire format to dump if non nil.
func dumpReq(req *http.Request, dump io.Writer) {
	if dump != nil {
		rd, err := http.DumpRequest(req, true)
		if err == nil {
			dump.Write(rd)
			dump.Write([]byte("\r\n\r\n--------------------------------------------------------------------------------------\r\n"))
			dump.Write([]byte("--------------------------------------------------------------------------------------\r\n\r\n\r\n"))
		} else {
			error("Cannot dump request: %s", err.String())
		}
	}
}

// Dump response in wire format to dump if non nil.
func dumpRes(res *http.Response, dump io.Writer) {
	if dump != nil {
		rd, err := http.DumpResponse(res, true)
		if err == nil {
			dump.Write(rd)
			dump.Write([]byte("\r\n\r\n======================================================================================\r\n"))
			dump.Write([]byte("======================================================================================\r\n\r\n\r\n"))
		} else {
			error("Cannot dump response: %s", err.String())
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
		trace("Cookie %s has MaxAge < 0.", cookie.Name)
		return false
	}

	if cookie.Expires.Year != 0 {
		if cookie.Expires.Seconds() < time.UTC().Seconds() {
			trace("Cookie %s has expired.", cookie.Name)
			return false
		}
	}

	trace("Cookie %s valid: MaxAge = %d, Expires = %s", cookie.Name, cookie.MaxAge, cookie.Expires.Format(http.TimeFormat))
	return true
}

// Take new cookies from recieved, and update/add to cookies 
func updateCookies(cookies []*http.Cookie, recieved []*http.Cookie) []*http.Cookie {
	trace("Updating list of %d cookies with %d fresh set cookies", len(cookies), len(recieved))
	// TODO: find solution with less allocations
	var update []*http.Cookie = make([]*http.Cookie, len(cookies))
	copy(update, cookies)

	for _, cookie := range recieved {
		trace("Cookie recieved: %s", cookie.String())
		// Prevent against bugs in http package which does not parse expires and maxage field properly
		for _, up := range cookie.Unparsed {
			if strings.HasPrefix(strings.ToLower(up), "expires=") && len(up) > 10 {
				val := up[8:]
				exptime, err := time.Parse(time.RFC1123, val)
				if err == nil {
					cookie.Expires = *exptime
				}
			}
			if strings.HasPrefix(strings.ToLower(up), "maxage=") && len(up) > 7 {
				ma, err := strconv.Atoi(up[7:])
				if err == nil {
					cookie.MaxAge = ma
				}
			}
		}

		isValid := valid(cookie)
		if !isValid {
			trace("Invalid cookie %s.", cookie.Name)
			continue
		}

		trace("Adding cookie %v", *cookie)
		update = append(update, cookie)
	}
	return update
}


// Test weather to send cookie to the given url.
func shouldSend(cookie *http.Cookie, u *url.URL) bool {
	if cookie.Secure && u.Scheme != "https" {
		trace("Wont send secure cookie to " + u.Scheme)
		return false
	}
	if cookie.HttpOnly && !(u.Scheme == "https" || u.Scheme == "http") {
		trace("Wont send HttpOnly cookie to " + u.Scheme)
		return false
	}
	if cookie.Expires.Year > 0 && cookie.Expires.Seconds() >= time.UTC().Seconds() {
		trace("Wont send expired cookie.")
		return false
	}
	if cookie.Path != "" && !strings.HasPrefix(u.Path, cookie.Path) {
		trace("Wont send " + cookie.Path + " cookie to " + u.Path)
		return false
	}

	// We do allow toplevel wildcard domains like .org :-)
	if cookie.Domain != "" {
		if cookie.Domain[0] == '.' {
			if !strings.HasSuffix(u.Host, cookie.Domain) {
				trace("Wont send wildcard " + cookie.Domain + " cookie to " + u.Host)
				return false
			}
		} else if u.Host != cookie.Domain {
			trace("Wont send " + cookie.Domain + " cookie to " + u.Host)
			return false
		}
	}
	return true
}


// A client which does not follow any redirects.
var nonfollowingClient http.Client = http.Client{
    Transport: nil, 
    CheckRedirect: func(req *http.Request, via []*http.Request) os.Error {
		if len(via) > 0 {
			return os.NewError("WE DONT FOLLOW")
		}
		return nil
	},
}


func redirectChecker(req *http.Request, via []*http.Request) os.Error {
	if len(via) >= 10 {
		return os.NewError("stopped after 10 redirects")
	}
	return nil
}


// Perform the request and follow up to 10 redirects.
// All cookie setting are collected, the final URL is reported.
func DoAndFollow(ireq *http.Request, t *Test) (r *http.Response, finalUrl string, cookies []*http.Cookie, err os.Error) {
	info("%s %s", ireq.Method, ireq.URL.String())

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
			for _, cookie := range cookies {
				if !shouldSend(cookie, req.URL) {
					trace("Skipped cookie %s.", cookie)
				} else {
					trace("Adding cookie to request in redirect: %s", cookie)
					req.AddCookie(&http.Cookie{Name: cookie.Name, Value: cookie.Value})
				}
			}

		}
		dumpReq(req, t.Dump)
		urlStr = req.URL.String()
		if r, err = nonfollowingClient.Do(req); err != nil {
			if strings.HasSuffix(err.String(), "WE DONT FOLLOW") {
				err = nil
			} else {
				return
			}
		}
		dumpRes(r, t.Dump)

		finalUrl = r.Request.URL.String()
		cookies = updateCookies(cookies, r.Cookies())

		if shouldRedirect(r.StatusCode) {
			r.Body.Close()
			if urlStr = r.Header.Get("Location"); urlStr == "" {
				err = os.NewError(fmt.Sprintf("%d response missing Location header", r.StatusCode))
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
func Get(t *Test) (r *http.Response, finalUrl string, cookies []*http.Cookie, err os.Error) {
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

	debug("Will get from %s", req.URL.String())
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
			trace("File to upload present.")
			return true
		}
	}
	return false
}

// allowed characters in a multipart boundary and own random numner generator
var multipartChars []byte = []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var boundaryRand *rand.Rand = rand.New(rand.NewSource(time.Seconds()))

// Consruct a new random boundary for multipart messages
func newBoundary() string {
	n := 15 + boundaryRand.Intn(20)
	b := [60]byte{}
	for i := 0; i < 60-n; i++ {
		b[i] = '-'
	}
	for i := 60 - n; i < 60; i++ {
		b[i] = multipartChars[boundaryRand.Intn(46)]
	}
	boundary := string(b[:])
	trace("New boundary: %s", boundary)
	return boundary
}

// Format the parameter map as a multipart message body.
func multipartBody(param *map[string][]string) (*bytes.Buffer, string) {
	var body *bytes.Buffer = &bytes.Buffer{}
	var boundary = newBoundary()

	var mpwriter = multipart.NewWriter(body)

	// All non-file parameters come first
	for n, v := range *param {
		if len(v) > 0 && strings.HasPrefix(v[0], "@file:") {
			continue // files go at the end
		}
		if true || len(v) > 0 {
			for _, vv := range v {
				trace("Added parameter %s with value '%s' to request body.", n, vv)
				mpwriter.WriteField(n, vv)
			}
		} else {
			trace("Adding empty parameter %s to request body.", n)
			var part = fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\n\r\n\r\n", boundary, n)
			body.WriteString(part)
		}
	}

	// File parameters at bottom
	for n, v := range *param {
		if !(len(v) > 0 && strings.HasPrefix(v[0], "@file:")) {
			continue // allready written
		}
		filename := v[0][6:]
		trace("Adding file '%s' as %s to request body.", filename, n)
		var ct string = "application/octet-stream"
		if i := strings.LastIndex(filename, "."); i != -1 {
			ct = mime.TypeByExtension(filename[i:])
			if ct == "" {
				ct = "application/octet-stream"
			}
		}

		basename := path.Base(filename)
		fw, err := mpwriter.CreateFormFile(n, basename)
		if err != nil {
			warn("Cannot write file multipart: ", err.String())
			continue
		}

		file, err := os.Open(filename)
		defer file.Close()
		if err != nil {
			warn("Cannot read from file '%s': %s.", filename, err.String())
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
func Post(t *Test) (r *http.Response, finalUrl string, cookies []*http.Cookie, err os.Error) {
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

	debug("Will post to %s", req.URL.String())

	r, finalUrl, cookies, err = DoAndFollow(req, t)
	return
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() os.Error { return nil }
