package networking

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerListeningAddress(t *testing.T) {
    server := Server{Host: net.IPv4(127, 0, 0, 1), Port: 443}
	assert.Equal(t, "127.0.0.1:443", server.ListeningAddress(), "Server Url should be 127.0.0.1:443")
}