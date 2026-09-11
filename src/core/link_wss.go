package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/Arceliar/phony"
	"github.com/coder/websocket"
)

type linkWSS struct {
	phony.Inbox
	*links
	tlsconfig *tls.Config
}

type linkWSSConn struct {
	net.Conn
}

func (l *links) newLinkWSS() *linkWSS {
	lwss := &linkWSS{
		links:     l,
		tlsconfig: l.core.config.tls.Clone(),
	}
	return lwss
}

func (l *linkWSS) dial(ctx context.Context, url *url.URL, info linkInfo, options linkOptions) (net.Conn, error) {
	// WSS terminates at an HTTPS server, often a reverse proxy. Authenticate
	// its DNS identity with system trust, not the mesh self-signed policy.
	tlsconfig := &tls.Config{MinVersion: tls.VersionTLS12}
	return l.findSuitableIP(url, func(hostname string, ip net.IP, port int) (net.Conn, error) {
		tlsconfig.ServerName = hostname
		tlsconfig.MinVersion = tls.VersionTLS12
		tlsconfig.MaxVersion = tls.VersionTLS13
		u := *url
		u.Host = net.JoinHostPort(ip.String(), fmt.Sprintf("%d", port))
		addr := &net.TCPAddr{
			IP:   ip,
			Port: port,
		}
		dialer, err := l.tcp.dialerFor(addr, info.sintf)
		if err != nil {
			return nil, err
		}
		wsconn, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
			HTTPClient: &http.Client{
				Transport: &http.Transport{
					Proxy:           http.ProxyFromEnvironment,
					DialContext:     dialer.DialContext,
					TLSClientConfig: tlsconfig,
				},
			},
			Subprotocols: []string{"ygg-ws", "uqda-ws"},
			Host:         hostname,
		})
		if err != nil {
			return nil, err
		}
		return &linkWSSConn{
			Conn: websocket.NetConn(ctx, wsconn, websocket.MessageBinary),
		}, nil
	})
}

func (l *linkWSS) listen(ctx context.Context, url *url.URL, _ string) (net.Listener, error) {
	return nil, fmt.Errorf("WSS listener not supported, use WS listener behind reverse proxy instead")
}
