package admin

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

const (
	DoctorPass = "pass"
	DoctorWarn = "warning"
	DoctorFail = "failure"
)

type DoctorRequest struct{}

type DoctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

type DoctorResponse struct {
	Status          string        `json:"status"`
	Checks          []DoctorCheck `json:"checks"`
	Recommendations []string      `json:"recommendations,omitempty"`
}

type doctorSnapshot struct {
	buildName      string
	buildVersion   string
	address        string
	routingEntries uint64
	adminListen    string
	adminMode      os.FileMode
	adminStatError error
	tunKnown       bool
	tunEnabled     bool
	tunName        string
	tunMTU         uint64
	peers          []PeerEntry
	multicastKnown bool
	multicastCount int
}

type doctorTUNResponse struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name,omitempty"`
	MTU     uint64 `json:"mtu,omitempty"`
}

type doctorMulticastResponse struct {
	Interfaces []json.RawMessage `json:"multicast_interfaces"`
}

func (a *AdminSocket) doctorHandler(_ *DoctorRequest, res *DoctorResponse) error {
	self := &GetSelfResponse{}
	if err := a.getSelfHandler(&GetSelfRequest{}, self); err != nil {
		return fmt.Errorf("inspect node identity: %w", err)
	}
	peers := &GetPeersResponse{}
	if err := a.getPeersHandler(&GetPeersRequest{SortBy: "latency"}, peers); err != nil {
		return fmt.Errorf("inspect peers: %w", err)
	}

	snapshot := doctorSnapshot{
		buildName:      self.BuildName,
		buildVersion:   self.BuildVersion,
		address:        self.IPAddress,
		routingEntries: self.RoutingEntries,
		adminListen:    string(a.config.listenaddr),
		peers:          peers.Peers,
	}

	if u, err := url.Parse(snapshot.adminListen); err == nil && strings.EqualFold(u.Scheme, "unix") && u.Path != "" && !strings.HasPrefix(u.Path, "@") {
		if info, statErr := os.Stat(u.Path); statErr == nil {
			snapshot.adminMode = info.Mode().Perm()
		} else {
			snapshot.adminStatError = statErr
		}
	}

	var tun doctorTUNResponse
	if ok, err := a.callDoctorDependency("gettun", &tun); err != nil {
		return fmt.Errorf("inspect TUN: %w", err)
	} else if ok {
		snapshot.tunKnown = true
		snapshot.tunEnabled = tun.Enabled
		snapshot.tunName = tun.Name
		snapshot.tunMTU = tun.MTU
	}

	var multicast doctorMulticastResponse
	if ok, err := a.callDoctorDependency("getmulticastinterfaces", &multicast); err != nil {
		return fmt.Errorf("inspect multicast: %w", err)
	} else if ok {
		snapshot.multicastKnown = true
		snapshot.multicastCount = len(multicast.Interfaces)
	}

	*res = evaluateDoctor(snapshot)
	return nil
}

func (a *AdminSocket) callDoctorDependency(name string, out interface{}) (bool, error) {
	a.handlerMu.RLock()
	h, ok := a.handlers[name]
	a.handlerMu.RUnlock()
	if !ok {
		return false, nil
	}
	value, err := h.handler(json.RawMessage(`{}`))
	if err != nil {
		return true, err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return true, err
	}
	return true, json.Unmarshal(data, out)
}

func evaluateDoctor(snapshot doctorSnapshot) DoctorResponse {
	res := DoctorResponse{Status: DoctorPass}
	add := func(name, status, summary string) {
		res.Checks = append(res.Checks, DoctorCheck{Name: name, Status: status, Summary: summary})
		if doctorSeverity(status) > doctorSeverity(res.Status) {
			res.Status = status
		}
	}

	buildSummary := fmt.Sprintf("%s %s is running", snapshot.buildName, snapshot.buildVersion)
	if snapshot.buildName == "unknown" && snapshot.buildVersion == "unknown" {
		buildSummary = "a development build is running"
	}
	add("daemon", DoctorPass, buildSummary)
	if snapshot.address == "" {
		add("identity", DoctorFail, "the node has no derived IPv6 address")
		res.Recommendations = append(res.Recommendations, "Inspect the configured private key and restart UQDA.")
	} else {
		add("identity", DoctorPass, fmt.Sprintf("node address %s is available", snapshot.address))
	}

	evaluateDoctorAdmin(snapshot, &res, add)

	switch {
	case snapshot.tunKnown && snapshot.tunEnabled:
		add("tun", DoctorPass, fmt.Sprintf("interface %s is enabled with MTU %d", snapshot.tunName, snapshot.tunMTU))
	case snapshot.tunKnown:
		add("tun", DoctorWarn, "the TUN interface is disabled")
		res.Recommendations = append(res.Recommendations, "Enable TUN unless this node is intentionally router-only or embedded.")
	default:
		add("tun", DoctorWarn, "TUN diagnostics are unavailable")
	}

	connected := 0
	bestLatency := int64(0)
	for _, peer := range snapshot.peers {
		if !peer.Up {
			continue
		}
		connected++
		if latency := peer.Latency.Microseconds(); latency > 0 && (bestLatency == 0 || latency < bestLatency) {
			bestLatency = latency
		}
	}
	switch {
	case len(snapshot.peers) == 0:
		add("peers", DoctorWarn, "no peers are configured or discovered")
		res.Recommendations = append(res.Recommendations, "Add a trusted nearby peer or enable multicast discovery on the intended LAN.")
	case connected == 0:
		add("peers", DoctorWarn, fmt.Sprintf("0/%d peers are connected", len(snapshot.peers)))
		res.Recommendations = append(res.Recommendations, "Run 'sudo uqdactl getPeers' and inspect Last Error.")
	case bestLatency > 0:
		add("peers", DoctorPass, fmt.Sprintf("%d/%d connected; best direct RTT %.2fms", connected, len(snapshot.peers), float64(bestLatency)/1000))
	default:
		add("peers", DoctorPass, fmt.Sprintf("%d/%d connected; waiting for an RTT sample", connected, len(snapshot.peers)))
	}

	if snapshot.routingEntries <= 1 {
		add("routing", DoctorWarn, fmt.Sprintf("routing table has %d entry; only this node may be reachable", snapshot.routingEntries))
		res.Recommendations = append(res.Recommendations, "Wait for routing convergence after a peer connects, then run doctor again.")
	} else {
		add("routing", DoctorPass, fmt.Sprintf("routing table has %d entries", snapshot.routingEntries))
	}

	if snapshot.multicastKnown {
		if snapshot.multicastCount > 0 {
			add("multicast", DoctorPass, fmt.Sprintf("discovery is active on %d interface(s)", snapshot.multicastCount))
		} else if len(snapshot.peers) == 0 {
			add("multicast", DoctorWarn, "no active discovery interfaces and no configured peers")
		} else {
			add("multicast", DoctorPass, "discovery is disabled; configured peers provide bootstrap connectivity")
		}
	}

	return res
}

func evaluateDoctorAdmin(snapshot doctorSnapshot, res *DoctorResponse, add func(string, string, string)) {
	u, err := url.Parse(snapshot.adminListen)
	if err != nil {
		add("admin", DoctorFail, "the administration endpoint is malformed")
		res.Recommendations = append(res.Recommendations, "Set AdminListen to a protected Unix socket or a loopback-only TCP endpoint.")
		return
	}
	switch strings.ToLower(u.Scheme) {
	case "unix":
		if strings.HasPrefix(u.Path, "@") {
			add("admin", DoctorPass, "the administration endpoint uses an abstract local Unix socket")
			return
		}
		if snapshot.adminStatError != nil {
			add("admin", DoctorFail, "the administration socket could not be inspected")
			res.Recommendations = append(res.Recommendations, "Verify the admin socket path and service permissions.")
			return
		}
		if snapshot.adminMode != 0600 {
			add("admin", DoctorFail, fmt.Sprintf("the administration socket mode is %04o; expected 0600", snapshot.adminMode))
			res.Recommendations = append(res.Recommendations, "Restrict the admin socket to mode 0600 and restart UQDA.")
			return
		}
		add("admin", DoctorPass, "the administration socket is restricted to mode 0600")
	case "tcp":
		host := u.Hostname()
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() || strings.EqualFold(host, "localhost") {
			add("admin", DoctorPass, "the TCP administration endpoint is restricted to loopback")
			return
		}
		add("admin", DoctorFail, "the unauthenticated TCP administration endpoint is not restricted to loopback")
		res.Recommendations = append(res.Recommendations, "Immediately bind AdminListen to localhost or use a mode-0600 Unix socket.")
	default:
		add("admin", DoctorFail, "the administration endpoint uses an unsupported scheme")
		res.Recommendations = append(res.Recommendations, "Use a protected Unix socket or a loopback-only TCP endpoint.")
	}
}

func doctorSeverity(status string) int {
	switch status {
	case DoctorFail:
		return 2
	case DoctorWarn:
		return 1
	default:
		return 0
	}
}
