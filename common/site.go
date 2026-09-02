package common

import "strings"

// ASR service sites. The empty value and SiteCN are the China (domestic)
// cluster; SiteIntl is the international cluster. The two clusters use
// different hostnames and typically different TRTC credentials.
const (
	SiteCN   = "cn"
	SiteIntl = "intl"

	HostCN   = "asr.cloud-rtc.com"
	HostIntl = "asr-intl.cloud-rtc.com"
)

// HostForSite returns the ASR hostname for site. An empty site is the
// domestic cluster. Unknown values return ErrCodeInvalidParam.
func HostForSite(site string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(site)) {
	case "", SiteCN:
		return HostCN, nil
	case SiteIntl:
		return HostIntl, nil
	default:
		return "", NewASRErrorf(ErrCodeInvalidParam,
			`unsupported site %q, want %q or %q`, site, SiteCN, SiteIntl)
	}
}

// WSEndpoint returns the realtime WebSocket origin for site (no path).
func WSEndpoint(site string) (string, error) {
	host, err := HostForSite(site)
	if err != nil {
		return "", err
	}
	return "wss://" + host, nil
}

// HTTPEndpoint returns the sentence/file HTTPS origin for site (no path).
func HTTPEndpoint(site string) (string, error) {
	host, err := HostForSite(site)
	if err != nil {
		return "", err
	}
	return "https://" + host, nil
}

// ResolveWSEndpoint returns override when it is non-empty, otherwise the
// site-derived realtime origin.
func ResolveWSEndpoint(override, site string) (string, error) {
	if override != "" {
		return override, nil
	}
	return WSEndpoint(site)
}

// ResolveHTTPEndpoint returns override when it is non-empty, otherwise the
// site-derived HTTPS origin.
func ResolveHTTPEndpoint(override, site string) (string, error) {
	if override != "" {
		return override, nil
	}
	return HTTPEndpoint(site)
}

// SiteOf returns the credential's site, or "" when credential is nil.
func SiteOf(c *Credential) string {
	if c == nil {
		return ""
	}
	return c.Site
}
