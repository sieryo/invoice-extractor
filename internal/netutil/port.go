package netutil

import (
	"fmt"
	"net"
)

func FindAvailablePort(start, max int) (int, error) {
	for port := start; port <= max; port++ {
		addr := fmt.Sprintf(":%d", port)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			continue
		}
		_ = ln.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no available port between %d-%d", start, max)
}
