package common

import (
	"errors"
	"testing"
)

func TestHostForSite(t *testing.T) {
	cases := []struct {
		site string
		host string
		fail bool
	}{
		{site: "", host: HostCN},
		{site: SiteCN, host: HostCN},
		{site: "CN", host: HostCN},
		{site: " cn ", host: HostCN},
		{site: SiteIntl, host: HostIntl},
		{site: "INTL", host: HostIntl},
		{site: "mars", fail: true},
	}
	for _, tc := range cases {
		host, err := HostForSite(tc.site)
		if tc.fail {
			if err == nil {
				t.Fatalf("HostForSite(%q) succeeded, want error", tc.site)
			}
			var asrErr *ASRError
			if !errors.As(err, &asrErr) || asrErr.Code != ErrCodeInvalidParam {
				t.Fatalf("HostForSite(%q) error = %v, want invalid param", tc.site, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("HostForSite(%q) error = %v", tc.site, err)
		}
		if host != tc.host {
			t.Fatalf("HostForSite(%q) = %q, want %q", tc.site, host, tc.host)
		}
	}
}

func TestResolveEndpointsHonorOverrideAndSite(t *testing.T) {
	ws, err := ResolveWSEndpoint("", SiteIntl)
	if err != nil {
		t.Fatal(err)
	}
	if ws != "wss://"+HostIntl {
		t.Fatalf("intl ws = %q", ws)
	}

	httpEP, err := ResolveHTTPEndpoint("", "")
	if err != nil {
		t.Fatal(err)
	}
	if httpEP != "https://"+HostCN {
		t.Fatalf("default http = %q", httpEP)
	}

	over, err := ResolveWSEndpoint("wss://mock.local", SiteIntl)
	if err != nil {
		t.Fatal(err)
	}
	if over != "wss://mock.local" {
		t.Fatalf("override lost: %q", over)
	}
}

func TestCredentialSetSite(t *testing.T) {
	c := NewCredential(1, 2, "k")
	if SiteOf(c) != "" {
		t.Fatalf("default site = %q, want empty", SiteOf(c))
	}
	c.SetSite(SiteIntl)
	if c.Site != SiteIntl {
		t.Fatalf("site = %q, want %q", c.Site, SiteIntl)
	}
}
