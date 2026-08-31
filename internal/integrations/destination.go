package integrations

import (
	"net"
	"net/url"
	"strings"
)

// SafeDestination rejects credentials, local names, and any private or
// reserved address returned by DNS. Dispatchers re-check it to limit DNS
// rebinding between endpoint creation and delivery.
func SafeDestination(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || strings.EqualFold(parsed.Hostname(), "localhost") || parsed.User != nil {
		return false
	}
	if address := net.ParseIP(parsed.Hostname()); address != nil {
		return publicAddress(address)
	}
	addresses, err := net.LookupIP(parsed.Hostname())
	if err != nil || len(addresses) == 0 {
		return false
	}
	for _, address := range addresses {
		if !publicAddress(address) {
			return false
		}
	}
	return true
}

func publicAddress(address net.IP) bool {
	return address.IsGlobalUnicast() && !address.IsPrivate() && !address.IsLoopback() && !address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast()
}
