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
	"rand"
	"time"
	"path"
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
	var update []*http.Cookie = make([]*http.Cookie, 0)

	for _, cookie := range recieved {
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

		if !valid(cookie) {
			update = append(update, &http.Cookie{Name: cookie.Name, Value: cookie.Value, MaxAge: -999})
			continue
		}

		var found bool
		for _, c := range cookies {
			if c.Name == cookie.Name {
				trace("Overwriting existing cookie %s with new Value %s.", cookie.Name, cookie.Value)
				update = append(update, &http.Cookie{Name: cookie.Name, Value: cookie.Value})
				break
			}
		}
		if !found && valid(cookie) {
			trace("Adding new cookie %s with value %s.", cookie.Name, cookie.Value)
			update = append(update, cookie)
		}

	}
	return update
}


// Perform the request and follow up to 10 redirects.
// All cookie setting are collected, the final URL is reported.
func DoAndFollow(req *http.Request, dump io.Writer) (response *http.Response, finalUrl string, cookies []*http.Cookie, err os.Error) {
	// TODO: set referrer header on redirects.

	/*
		// Move User-Agent from Header to Request
		if ua := req.UserAgent(); ua != "" {
			req.UserAgent = ua
			req.Header.Del("User-Agent")
		}
	*/

	info("%s %s", req.Method, req.URL.String())
	dumpReq(req, dump)
	response, err = http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	dumpRes(response, dump)

	finalUrl = response.Request.URL.String()
	cookies = updateCookies(cookies, response.Cookies())
	for _, c := range response.Cookies() {
		if _, err := req.Cookie(c.Name); err != nil {
			req.AddCookie(c)
		}
	}
	//	req.Cookie = updateCookies(req.Cookie, response.Cookies())


	if !shouldRedirect(response.StatusCode) {
		return
	}

	// TODO: will reach this point only for POST->redirect. Use DefaultClient here to handle 
	// the following redirects.

	// Start redirecting to final destination
	response.Body.Close()
	var base = req.URL

	// Following the redirect chain is done with a cleaned/empty GET request.
	req.Method = "GET"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Header.Del("Content-Type")
	req.Header.Del("Content-Length")
	req.Header.Del("Accept-Encoding")
	req.Header.Del("Connection")
	req.Body = nil
	for redirect := 0; redirect < 10; redirect++ {
		var url string

		if url = response.Header.Get("Location"); url == "" {
			fmt.Printf("Header:\n%v", response.Header)
			err = os.NewError(fmt.Sprintf("%d response missing Location header", response.StatusCode))
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
		dumpReq(req, dump)

		if response, err = http.DefaultClient.Do(req); err != nil {
			return
		}

		dumpRes(response, dump)
		finalUrl = url
		cookies = updateCookies(cookies, response.Cookies())
		for _, c := range response.Cookies() {
			if _, err := req.Cookie(c.Name); err != nil {
				req.AddCookie(c)
			}
		}
		// req.Cookie = updateCookies(req.Cookie, response.SetCookie)

		if !shouldRedirect(response.StatusCode) {
			return
		}
		response.Body.Close()
		base = req.URL

	}
	err = os.NewError("Too many redirects.")
	return
}

func urlencode(param map[string][]string) string {
	s := ""
	for k, vs := range param {
		for _, v := range vs {
			s += fmt.Sprintf("&%s=%s", k, http.URLEscape(v))
		}
	}
	if len(s) > 0 {
		s = s[1:]
	}
	return s
}

// Perform a GET request for the test t.
func Get(t *Test) (r *http.Response, finalUrl string, cookies []*http.Cookie, err os.Error) {
	var url = t.Url // <-- Patched

	var req http.Request
	req.Method = "GET"
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Header = http.Header{}

	if len(t.Param) > 0 {
		ep := urlencode(t.Param)
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
	r, finalUrl, cookies, err = DoAndFollow(&req, t.Dump)
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

	// All non-file parameters come first
	for n, v := range *param {
		if len(v) > 0 && strings.HasPrefix(v[0], "@file:") {
			continue
		}
		if len(v) > 0 {
			for _, vv := range v {
				trace("Added parameter %s with value '%s' to request body.", n, vv)
				var part = fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\n\r\n%s\r\n", boundary, n, vv)
				body.WriteString(part)
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
			continue
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
		var part = fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", boundary, n, basename)
		part += fmt.Sprintf("Content-Type: %s\r\n\r\n", ct)
		file, err := os.Open(filename)
		defer file.Close()
		if err != nil {
			warn("Cannot read from file '%s': %s.", filename, err.String())
			continue
		}
		body.WriteString(part)
		body.ReadFrom(file)
		body.WriteString("\r\n")
	}
	body.WriteString("--" + boundary + "--\r\n")

	return body, boundary
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
	if hasFile(&t.Param) || t.Method == "POST:mp" {
		var boundary string
		body, boundary = multipartBody(&t.Param)
		contentType = "multipart/form-data; boundary=" + boundary
	} else {
		contentType = "application/x-www-form-urlencoded"
		bodystr := urlencode(t.Param)
		body = bytes.NewBuffer([]byte(bodystr))
	}

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

	r, finalUrl, cookies, err = DoAndFollow(&req, t.Dump)
	return
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() os.Error { return nil }
