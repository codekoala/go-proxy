package loadbalancer

import (
	"github.com/yusing/go-proxy/internal/utils/strutils"
)

type Mode string

const (
	Unset      Mode = ""
	RoundRobin Mode = "roundrobin"
	LeastConn  Mode = "leastconn"
	IPHash     Mode = "iphash"
)

func (mode *Mode) ValidateUpdate() bool {
	switch strutils.ToLowerNoSnake(string(*mode)) {
	case "":
		return true
	case string(RoundRobin):
		*mode = RoundRobin
		return true
	case string(LeastConn):
		*mode = LeastConn
		return true
	case string(IPHash):
		*mode = IPHash
		return true
	}
	*mode = RoundRobin
	return false
}
