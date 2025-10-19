package utils

import "github.com/ip2location/ip2location-go/v9"

func ValidateIp(ip string) bool {
	return ip2location.OpenTools().IsIPv4(ip) || ip2location.OpenTools().IsIPv6(ip)
}
