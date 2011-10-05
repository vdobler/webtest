package suite

//
// Some kind of cookie jar...
//

import (
	"http"
	"url"
	"strings"
	"sync"
	"time"
)

const (
	NO_TOPLEVEL_DOMAIN = iota
	SAME_OR_SUBDOMAIN
	STRICT_SAME_DOMAIN
)

// All cookies in the jar have a efective domain of the form ".www.domain.org"
// 
type CookieJar struct {
	cookies []*http.Cookie // all our cookies accessible by domain
	mutex   sync.Mutex
	policy  int
}

// NewCookieJar sets up a new empty cookie jar
func NewCookieJar() *CookieJar {
	cj := &CookieJar{}
	cj.cookies = make([]*http.Cookie, 0, 10)
	return cj
}

// Copy returns a new cookie jar with the same cookies.
func (jar *CookieJar) Copy() *CookieJar {
	cj := &CookieJar{}
	jar.mutex.Lock()
	n := len(jar.cookies)
	cj.cookies = make([]*http.Cookie, n)
	for i := 0; i < n; i++ {
		nc := *jar.cookies[i]
		cj.cookies[i] = &nc
	}
	jar.mutex.Unlock()
	return cj
}

// Check if cookie was requested to be deleted or is expired.
func expiredOrDeleted(c *http.Cookie) bool {
	if c.MaxAge < 0 {
		return true
	}

	if c.Expires.Year == 0 {
		return false
	}

	return c.Expires.Seconds() >= time.UTC().Seconds()
}

// Find looks up the index of cookie in our cookie jar.
// The lookup is based on an exact match of the (Name, Domain, Path) tripple.
func (jar *CookieJar) find(domain, path, name string) (int, *http.Cookie) {
	domain = stripPort(domain)
	for i, c := range jar.cookies {
		if c.Domain == domain && path == c.Path && name == c.Name {
			return i, c
		}
	}
	return -1, nil
}

// Contains looks up the cookie in our cookie jar and returns nil if not found.
// The lookup is based on an exact match of the (Name, Domain, Path) tripple.
func (jar *CookieJar) Contains(domain, path, name string) *http.Cookie {
	_, c := jar.find(domain, path, name)
	return c
}

func (jar *CookieJar) All() []*http.Cookie {
	return jar.cookies
}

// Update will update (add new, update existing or remove deleted) the jar 
// with the given cookie as recieved from domain.
// It's a method on CookieJar to apply the jars policy in the future.
func (jar *CookieJar) Update(cookie http.Cookie, domain string) {
	domain = stripPort(domain)
	trace("Update cookie %s:%s:%s=%s for domain %s",
		cookie.Name, cookie.Domain, cookie.Path, cookie.Value, domain)

	// make sure Domain is set (and starts with '.' and Path is set
	// TODO: prevent stuff like .net or .co.uk ....
	if cookie.Domain == "" {
		if domain == "" {
			panic("Not both empty")
		}
		cookie.Domain = domain
	}
	if cookie.Domain[0] != '.' {
		cookie.Domain = "." + cookie.Domain
	}
	if cookie.Path == "" {
		cookie.Path = "/"
	}

	// Set Expires from MaxAge if set
	if cookie.MaxAge > 0 {
		cookie.Expires = *time.SecondsToLocalTime(time.LocalTime().Seconds() + int64(cookie.MaxAge))
	}

	trace("Prepared cookie for update %v", cookie)

	jar.mutex.Lock()
	defer jar.mutex.Unlock()

	idx, _ := jar.find(cookie.Domain, cookie.Path, cookie.Name)
	trace("Cookie allready in jar: %t", idx != -1)

	if expiredOrDeleted(&cookie) {
		trace("Cookie is expired/deleted")
		if idx != -1 {
			jar.cookies = append(jar.cookies[:idx], jar.cookies[idx+1:]...)
		}
		return
	}

	if idx == -1 { // new cookie
		jar.cookies = append(jar.cookies, &cookie)
	} else { // update
		jar.cookies[idx] = &cookie
	}

}

// check if current host of request matches effective domain of cookie
func (jar *CookieJar) domainMatch(host, domain string) bool {
	if host == domain[1:] { // www.host.com = .www.host.com
		return true
	}
	if strings.HasSuffix(host, domain) { // sso.host.com = .host.com
		return true
	}
	return false
}

// Check if path matches
func pathMatch(reqpath, cookiepath string) bool {
	if strings.HasPrefix(reqpath, cookiepath) { // /some/path matches /some
		return true
	}
	return false
}

func stripPort(host string) string {
	e := strings.Split(host, ":")
	return e[0]
}

// Select selects all cookies which should be sent to the given URL u.
func (jar *CookieJar) Select(u *url.URL) (cookies []*http.Cookie) {
	host := stripPort(u.Host)
	path := u.Path
	trace("Select cookie for %s%s", host, path)

	// list of possible cookies
	list := make([]*http.Cookie, 0, 5)
	tbd := make([]int, 0, 2) // Expired Cookies
	for i, c := range jar.cookies {
		trace("  try cookie %v", c)
		if !jar.domainMatch(host, c.Domain) {
			trace("    wrong domain %s", c.Domain)
			continue
		}
		if !pathMatch(path, c.Path) {
			trace("    wrong path %s", c.Path)
			continue
		}
		if c.HttpOnly && !(u.Scheme == "http" || u.Scheme == "https") {
			trace("    wrong protocol %s for HttpOnly", u.Scheme)
			continue
		}
		if c.Secure && u.Scheme != "https" {
			trace("    not secure protocol %s for Secure cookie", u.Scheme)
			continue
		}
		if expiredOrDeleted(c) {
			tbd = append(tbd, i)
		}
		trace("    --> okay")
		list = append(list, c)
	}

	// Remove expired cookies from list
	if len(tbd) > 0 {
		trace("Will remove %d expired cookies from jar.", len(tbd))
		jar.mutex.Lock()
		for i, idx := range tbd {
			jar.cookies = append(jar.cookies[:idx-i], jar.cookies[idx+1-i:]...)
		}
		jar.mutex.Unlock()
	}

	// map of name to cookie to send
	m := make(map[string]*http.Cookie)
	for _, c := range list {
		if ac, ok := m[c.Name]; ok {
			trace("Same-name-cookie: %s", c.Name)
			if len(c.Path) > len(ac.Path) {
				// this one is more specific and should be used
				m[c.Name] = c
				trace("  use more specific %v", c)
			}
		} else {
			m[c.Name] = c
		}
	}

	// fill to cookies
	cookies = make([]*http.Cookie, len(m))
	i := 0
	for _, c := range m {
		cookies[i] = c
		i++
	}
	return
}
