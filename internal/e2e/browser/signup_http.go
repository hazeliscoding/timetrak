//go:build browser

package browser

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/http/cookiejar"
)

func newSignupHTTPClient(jar *cookiejar.Jar) *http.Client {
	return &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Stop at the 303 from /signup so we can read cookies.
			return http.ErrUseLastResponse
		},
	}
}

func randSuffix() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
