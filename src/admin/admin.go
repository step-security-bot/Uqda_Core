package admin

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"sort"
	"sync"

	"strings"
	"time"

	"github.com/Uqda/Core/src/core"
)

type AdminSocket struct {
	core     *core.Core
	log      core.Logger
	listener net.Listener
	handlers map[string]handler
	handlerMu sync.RWMutex
	connections chan struct{}
	done     chan struct{}
	config   struct {
		listenaddr ListenAddress
		tlsConfig *tls.Config
	}
}

type AdminSocketRequest struct {
	Name      string          `json:"request"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
	KeepAlive bool            `json:"keepalive,omitempty"`
}

type AdminSocketResponse struct {
	Status   string             `json:"status"`
	Error    string             `json:"error,omitempty"`
	Request  AdminSocketRequest `json:"request"`
	Response json.RawMessage    `json:"response"`
}

type handler struct {
	desc    string              // What does the endpoint do?
	args    []string            // List of human-readable argument names
	handler core.AddHandlerFunc // First is input map, second is output
}

type ListResponse struct {
	List []ListEntry `json:"list"`
}

type ListEntry struct {
	Command     string   `json:"command"`
	Description string   `json:"description"`
	Fields      []string `json:"fields,omitempty"`
}

// AddHandler is called for each admin function to add the handler and help documentation to the API.
func (a *AdminSocket) AddHandler(name, desc string, args []string, handlerfunc core.AddHandlerFunc) error {
	a.handlerMu.Lock()
	defer a.handlerMu.Unlock()
	if _, ok := a.handlers[strings.ToLower(name)]; ok {
		return errors.New("handler already exists")
	}
	a.handlers[strings.ToLower(name)] = handler{
		desc:    desc,
		args:    args,
		handler: handlerfunc,
	}
	return nil
}

// Init runs the initial admin setup.
func New(c *core.Core, log core.Logger, opts ...SetupOption) (*AdminSocket, error) {
	a := &AdminSocket{
		core:     c,
		log:      log,
		handlers: make(map[string]handler),
		connections: make(chan struct{}, 64),
	}
	for _, opt := range opts {
		a._applyOption(opt)
	}
	if a.config.listenaddr == "none" || a.config.listenaddr == "" {
		return nil, nil
	}

	listenaddr := string(a.config.listenaddr)
	u, err := url.Parse(listenaddr)
	if err == nil {
		switch strings.ToLower(u.Scheme) {
		case "unix":
			if _, err := os.Stat(u.Path); err == nil {
				a.log.Debugln("Admin socket", u.Path, "already exists, trying to clean up")
				_, dialErr := net.DialTimeout("unix", u.Path, time.Second*2)
				inUse := dialErr == nil
				if !inUse {
					var netErr net.Error
					if errors.As(dialErr, &netErr) && netErr.Timeout() {
						inUse = true
					}
				}
				if inUse {
					a.log.Errorln("Admin socket", u.Path, "already exists and is in use by another process")
					os.Exit(1)
				} else {
					if err := os.Remove(u.Path); err == nil {
						a.log.Debugln(u.Path, "was cleaned up")
					} else {
						a.log.Errorln(u.Path, "already exists and was not cleaned up:", err)
						os.Exit(1)
					}
				}
			}
			a.listener, err = net.Listen("unix", u.Path)
			if err == nil {
				abstract := u.Path != "" && u.Path[0] == '@'
				if !abstract {
					if err := secureUnixSocket(u.Path); err != nil {
						_ = a.listener.Close()
						return nil, fmt.Errorf("failed to secure admin socket %s: %w", u.Path, err)
					}
				}
			}
		case "tcp", "tls":
			a.listener, err = net.Listen("tcp", u.Host)
		default:
			a.listener, err = net.Listen("tcp", listenaddr)
		}
	} else {
		a.listener, err = net.Listen("tcp", listenaddr)
	}
	if err != nil {
		a.log.Errorf("Admin socket failed to listen: %v", err)
		os.Exit(1)
	}
	// TCP is authenticated on every platform. The transport label tcp:// is
	// retained for existing configurations; updated clients negotiate TLS.
	if a.listener.Addr().Network() == "tcp" {
		if a.config.tlsConfig == nil {
			_ = a.listener.Close()
			return nil, fmt.Errorf("TCP administration requires a local certificate")
		}
		a.listener = tls.NewListener(a.listener, a.config.tlsConfig)
	}
	a.log.Infof("%s admin socket listening on %s",
		strings.ToUpper(a.listener.Addr().Network()),
		a.listener.Addr().String())

	_ = a.AddHandler("list", "List available commands", []string{}, func(_ json.RawMessage) (interface{}, error) {
		a.handlerMu.RLock()
		defer a.handlerMu.RUnlock()
		res := &ListResponse{}
		for name, handler := range a.handlers {
			res.List = append(res.List, ListEntry{
				Command:     name,
				Description: handler.desc,
				Fields:      handler.args,
			})
		}
		sort.SliceStable(res.List, func(i, j int) bool {
			return strings.Compare(res.List[i].Command, res.List[j].Command) < 0
		})
		return res, nil
	})
	a.done = make(chan struct{})
	go a.listen()
	return a, a.core.SetAdmin(a)
}

func secureUnixSocket(path string) error {
	return os.Chmod(path, 0600)
}

func (a *AdminSocket) SetupAdminHandlers() {
	_ = a.AddHandler(
		"getSelf", "Show details about this node", []string{},
		func(in json.RawMessage) (interface{}, error) {
			req := &GetSelfRequest{}
			res := &GetSelfResponse{}
			if err := json.Unmarshal(in, &req); err != nil {
				return nil, err
			}
			if err := a.getSelfHandler(req, res); err != nil {
				return nil, err
			}
			return res, nil
		},
	)
	_ = a.AddHandler(
		"getPeers", "Show directly connected peers", []string{"sort"},
		func(in json.RawMessage) (interface{}, error) {
			req := &GetPeersRequest{}
			res := &GetPeersResponse{}
			if err := json.Unmarshal(in, &req); err != nil {
				return nil, err
			}
			if err := a.getPeersHandler(req, res); err != nil {
				return nil, err
			}
			return res, nil
		},
	)
	_ = a.AddHandler(
		"doctor", "Run safe node health and security diagnostics", []string{},
		func(in json.RawMessage) (interface{}, error) {
			req := &DoctorRequest{}
			res := &DoctorResponse{}
			if err := json.Unmarshal(in, req); err != nil {
				return nil, err
			}
			if err := a.doctorHandler(req, res); err != nil {
				return nil, err
			}
			return res, nil
		},
	)
	_ = a.AddHandler(
		"getTree", "Show known Tree entries", []string{},
		func(in json.RawMessage) (interface{}, error) {
			req := &GetTreeRequest{}
			res := &GetTreeResponse{}
			if err := json.Unmarshal(in, &req); err != nil {
				return nil, err
			}
			if err := a.getTreeHandler(req, res); err != nil {
				return nil, err
			}
			return res, nil
		},
	)
	_ = a.AddHandler(
		"getPaths", "Show established paths through this node", []string{},
		func(in json.RawMessage) (interface{}, error) {
			req := &GetPathsRequest{}
			res := &GetPathsResponse{}
			if err := json.Unmarshal(in, &req); err != nil {
				return nil, err
			}
			if err := a.getPathsHandler(req, res); err != nil {
				return nil, err
			}
			return res, nil
		},
	)
	_ = a.AddHandler(
		"getSessions", "Show established traffic sessions with remote nodes", []string{},
		func(in json.RawMessage) (interface{}, error) {
			req := &GetSessionsRequest{}
			res := &GetSessionsResponse{}
			if err := json.Unmarshal(in, &req); err != nil {
				return nil, err
			}
			if err := a.getSessionsHandler(req, res); err != nil {
				return nil, err
			}
			return res, nil
		},
	)
	_ = a.AddHandler(
		"addPeer", "Add a peer to the peer list", []string{"uri", "interface"},
		func(in json.RawMessage) (interface{}, error) {
			req := &AddPeerRequest{}
			res := &AddPeerResponse{}
			if err := json.Unmarshal(in, &req); err != nil {
				return nil, err
			}
			if err := a.addPeerHandler(req, res); err != nil {
				return nil, err
			}
			return res, nil
		},
	)
	_ = a.AddHandler(
		"removePeer", "Remove a peer from the peer list", []string{"uri", "interface"},
		func(in json.RawMessage) (interface{}, error) {
			req := &RemovePeerRequest{}
			res := &RemovePeerResponse{}
			if err := json.Unmarshal(in, &req); err != nil {
				return nil, err
			}
			if err := a.removePeerHandler(req, res); err != nil {
				return nil, err
			}
			return res, nil
		},
	)
}

// IsStarted returns true if the module has been started.
func (a *AdminSocket) IsStarted() bool {
	select {
	case <-a.done:
		// Not blocking, so we're not currently running
		return false
	default:
		// Blocked, so we must have started
		return true
	}
}

// Stop will stop the admin API and close the socket.
func (a *AdminSocket) Stop() error {
	if a == nil {
		return nil
	}
	if a.listener != nil {
		select {
		case <-a.done:
		default:
			close(a.done)
		}
		return a.listener.Close()
	}
	return nil
}

// listen is run by start and manages API connections.
func (a *AdminSocket) listen() {
	defer a.listener.Close()
	for {
		conn, err := a.listener.Accept()
		if err == nil {
			select {
			case a.connections <- struct{}{}:
				go func() {
					defer func() { <-a.connections }()
					a.handleRequest(conn)
				}()
			default:
				_ = conn.Close()
			}
		} else {
			select {
			case <-a.done:
				// Not blocked, so we havent started or already stopped
				return
			default:
				// Blocked, so we're supposed to keep running
			}
		}
	}
}

// handleRequest calls the request handler for each request sent to the admin API.
func (a *AdminSocket) handleRequest(conn net.Conn) {
	// Bound each session, including incomplete JSON and idle TLS handshakes.
	decoder := json.NewDecoder(io.LimitReader(conn, 1<<20))
	decoder.DisallowUnknownFields()

	encoder := json.NewEncoder(conn)
	encoder.SetIndent("", "  ")

	defer conn.Close()

	for {
		if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
			return
		}
		var err error
		var buf json.RawMessage
		var req AdminSocketRequest
		var resp AdminSocketResponse
		req.Arguments = []byte("{}")
		if err := func() error {
			if err = decoder.Decode(&buf); err != nil {
				return fmt.Errorf("failed to find request")
			}
			if err = json.Unmarshal(buf, &req); err != nil {
				return fmt.Errorf("failed to unmarshal request")
			}
			resp.Request = req
			if req.Name == "" {
				return fmt.Errorf("no request specified")
			}
			reqname := strings.ToLower(req.Name)
			a.handlerMu.RLock()
			handler, ok := a.handlers[reqname]
			a.handlerMu.RUnlock()
			if !ok {
				return fmt.Errorf("unknown action '%s', try 'list' for help", reqname)
			}
			res, err := handler.handler(req.Arguments)
			if err != nil {
				return err
			}
			if resp.Response, err = json.Marshal(res); err != nil {
				return fmt.Errorf("failed to marshal response: %w", err)
			}
			resp.Status = "success"
			return nil
		}(); err != nil {
			resp.Status = "error"
			resp.Error = err.Error()
		}
		if err = encoder.Encode(resp); err != nil {
			a.log.Debugln("Encode error:", err)
		}
		if !req.KeepAlive {
			break
		} else {
			continue
		}
	}
}

type DataUnit uint64

func (d DataUnit) String() string {
	switch {
	case d >= 1024*1024*1024*1024:
		return fmt.Sprintf("%2.1fTB", float64(d)/1024/1024/1024/1024)
	case d >= 1024*1024*1024:
		return fmt.Sprintf("%2.1fGB", float64(d)/1024/1024/1024)
	case d >= 1024*1024:
		return fmt.Sprintf("%2.1fMB", float64(d)/1024/1024)
	case d >= 100:
		return fmt.Sprintf("%2.1fKB", float64(d)/1024)
	default:
		return fmt.Sprintf("%dB", d)
	}
}
