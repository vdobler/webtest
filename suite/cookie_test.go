package suite

import (
	"fmt"
	"testing"
	"http"
)

func TestJarUpdate(t *testing.T) {
	jar := NewCookieJar()
	
	jar.Update(http.Cookie{Name: "myCookie1", Value: "myValue1"}, "www.example.org")
	if len(jar.cookies)!=1 || jar.cookies[0].Name!="myCookie1" ||
		jar.cookies[0].Value!="myValue1" ||jar.cookies[0].Domain!=".www.example.org" ||
		jar.cookies[0].Path != "/" {
		t.Error(fmt.Sprintf("myCookie1: %v", jar.cookies[0]))
	}

	jar.Update(http.Cookie{Name: "myCookie2", Value: "myValue2", Path: "/some/path"}, "www.example.org")
	if len(jar.cookies)!=2 || jar.cookies[1].Name!="myCookie2" ||
		jar.cookies[1].Value!="myValue2" ||jar.cookies[1].Domain!=".www.example.org" ||
		jar.cookies[1].Path != "/some/path" {
		t.Error(fmt.Sprintf("myCookie2: %v", jar.cookies[1]))
	}

	jar.Update(http.Cookie{Name: "myCookie3", Value: "myValue3", Path: "/other", Domain: "sub.www.example.org"}, 
		"www.example.org")
	if len(jar.cookies)!=3 || jar.cookies[2].Name!="myCookie3" ||
		jar.cookies[2].Value!="myValue3" || jar.cookies[2].Domain!=".sub.www.example.org" ||
		jar.cookies[2].Path != "/other" {
		t.Error(fmt.Sprintf("myCookie3: %v", jar.cookies[2]))
	}

	jar.Update(http.Cookie{Name: "myCookieX", Value: "myValueX", MaxAge: -1 }, "www.example.org")
	if len(jar.cookies)!=3 {
		t.Error(fmt.Sprintf("myCookieX: "))
	}

	jar.Update(http.Cookie{Name: "myCookie2", Value: "", Path: "/some/path", MaxAge: -1 }, "www.example.org")
	if len(jar.cookies)!=2 || jar.cookies[1].Name!="myCookie3" ||
		jar.cookies[1].Value!="myValue3" ||jar.cookies[1].Domain!=".sub.www.example.org" ||
		jar.cookies[1].Path != "/other" {
		t.Error(fmt.Sprintf("myCookie3 on pos 2: %v", jar.cookies[1]))
	}
}