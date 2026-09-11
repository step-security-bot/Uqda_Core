package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"suah.dev/protect"

	"github.com/Uqda/Core/src/admin"
	"github.com/Uqda/Core/src/config"
	"github.com/Uqda/Core/src/core"
	"github.com/Uqda/Core/src/multicast"
	"github.com/Uqda/Core/src/tun"
	"github.com/Uqda/Core/src/version"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"
)

func main() {
	// read config, speak DNS/TCP and/or over a UNIX socket
	if err := protect.Pledge("stdio rpath inet unix dns"); err != nil {
		panic(err)
	}

	// makes sure we can use defer and still return an error code to the OS
	os.Exit(run())
}

func run() (exitCode int) {
	logbuffer := &bytes.Buffer{}
	logger := log.New(logbuffer, "", log.Flags())

	defer func() {
		if r := recover(); r != nil {
			logger.Println("Fatal error:", r)
			fmt.Print(logbuffer)
			exitCode = 1
		}
	}()

	cmdLineEnv := newCmdLineEnv()
	cmdLineEnv.parseFlagsAndArgs()
	style, err := newOutputStyle(cmdLineEnv.color, cmdLineEnv.injson, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "uqdactl:", err)
		return 1
	}

	if cmdLineEnv.ver {
		fmt.Println("Build name:", version.BuildName())
		fmt.Println("Build version:", version.BuildVersion())
		fmt.Println("To get the version number of the running UQDA node, run", os.Args[0], "getSelf")
		return 0
	}

	if len(cmdLineEnv.args) == 0 {
		flag.Usage()
		return 0
	}
	if err := requireAdministrator(); err != nil {
		fmt.Fprintln(os.Stderr, "uqdactl:", err)
		return 1
	}

	cmdLineEnv.setEndpoint(logger)

	var conn net.Conn
	u, err := url.Parse(cmdLineEnv.endpoint)
	if err == nil {
		switch strings.ToLower(u.Scheme) {
		case "unix":
			logger.Println("Connecting to UNIX socket", cmdLineEnv.endpoint[7:])
			conn, err = net.Dial("unix", cmdLineEnv.endpoint[7:])
		case "tcp", "tls":
			logger.Println("Connecting to TCP socket", u.Host)
			conn, err = net.DialTimeout("tcp", u.Host, 10*time.Second)
		default:
			logger.Println("Unknown protocol or malformed address - check your endpoint")
			err = errors.New("protocol not supported")
		}
	} else {
		logger.Println("Connecting to TCP socket", cmdLineEnv.endpoint)
		conn, err = net.DialTimeout("tcp", cmdLineEnv.endpoint, 10*time.Second)
	}
	if err != nil {
		fmt.Fprint(os.Stderr, logbuffer.String())
		fmt.Fprintln(os.Stderr, "uqdactl: could not connect to the administration endpoint; verify that UQDA is running:", err)
		return 1
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		fmt.Fprintln(os.Stderr, "uqdactl:", err)
		return 1
	}
	if conn.RemoteAddr().Network() == "tcp" {
		f, err := os.Open(cmdLineEnv.configFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "uqdactl: cannot read local node authentication configuration:", err)
			return 1
		}
		var cfg config.NodeConfig
		_, readErr := cfg.ReadFrom(f)
		_ = f.Close()
		if readErr != nil {
			fmt.Fprintln(os.Stderr, "uqdactl: invalid authentication configuration:", readErr)
			return 1
		}
		tlsConfig := admin.PinnedTLSConfig(cfg.Certificate)
		if tlsConfig == nil {
			fmt.Fprintln(os.Stderr, "uqdactl: invalid local node certificate")
			return 1
		}
		secured := tls.Client(conn, tlsConfig)
		if err := secured.Handshake(); err != nil {
			fmt.Fprintln(os.Stderr, "uqdactl: administration authentication failed; update daemon and client together:", err)
			return 1
		}
		conn = secured
	}

	// Configuration and socket setup are complete; retain only the promises
	// needed to exchange the administration request.
	if err := protect.Pledge("stdio"); err != nil {
		panic(err)
	}

	logger.Println("Connected")
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)
	send := &admin.AdminSocketRequest{}
	recv := &admin.AdminSocketResponse{}
	args := map[string]string{}
	for c, a := range cmdLineEnv.args {
		if c == 0 {
			if strings.HasPrefix(a, "-") {
				logger.Printf("Ignoring flag %s as it should be specified before other parameters\n", a)
				continue
			}
			logger.Printf("Sending request: %v\n", a)
			send.Name = a
			continue
		}
		tokens := strings.SplitN(a, "=", 2)
		switch {
		case len(tokens) == 1:
			logger.Println("Ignoring invalid argument:", a)
		default:
			args[tokens[0]] = tokens[1]
		}
	}
	if send.Arguments, err = json.Marshal(args); err != nil {
		panic(err)
	}
	if err := encoder.Encode(&send); err != nil {
		panic(err)
	}
	logger.Printf("Request sent")
	if err := decoder.Decode(&recv); err != nil {
		panic(err)
	}
	if recv.Status == "error" {
		if err := recv.Error; err != "" {
			fmt.Println("Admin socket returned an error:", err)
		} else {
			fmt.Println("Admin socket returned an error but didn't specify any error text")
		}
		return 1
	}
	if cmdLineEnv.injson {
		if json, err := json.MarshalIndent(recv.Response, "", "  "); err == nil {
			fmt.Println(string(json))
		}
		if strings.EqualFold(send.Name, "doctor") {
			var resp admin.DoctorResponse
			if err := json.Unmarshal(recv.Response, &resp); err != nil {
				panic(err)
			}
			return doctorExitCode(resp)
		}
		return 0
	}

	opts := []tablewriter.Option{
		tablewriter.WithRowAlignment(tw.AlignLeft),
		tablewriter.WithHeaderAlignment(tw.AlignCenter),
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithDebug(false),
	}
	if !cmdLineEnv.borders {
		opts = append(opts, tablewriter.WithRenderer(renderer.NewBlueprint(tw.Rendition{
			Borders: tw.BorderNone,
			Settings: tw.Settings{
				Lines:      tw.LinesNone,
				Separators: tw.SeparatorsNone,
			},
		})))
	}
	table := tablewriter.NewTable(os.Stdout, opts...)

	switch strings.ToLower(send.Name) {
	case "doctor":
		var resp admin.DoctorResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		table.Header(style.headers("Status", "Check", "Result"))
		for _, check := range resp.Checks {
			status := strings.ToUpper(check.Status)
			_ = table.Append([]string{style.status(status), style.label(check.Name), check.Summary})
		}
		for _, recommendation := range resp.Recommendations {
			_ = table.Append([]string{style.status("NEXT"), style.label("recommendation"), recommendation})
		}
		_ = table.Render()
		fmt.Println(style.strong("Overall status:"), style.status(strings.ToUpper(resp.Status)))
		return doctorExitCode(resp)

	case "list":
		var resp admin.ListResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		table.Header(style.headers("Command", "Arguments", "Description"))
		for _, entry := range resp.List {
			for i := range entry.Fields {
				entry.Fields[i] = entry.Fields[i] + "=..."
			}
			_ = table.Append([]string{style.label(entry.Command), strings.Join(entry.Fields, ", "), entry.Description})
		}
		_ = table.Render()

	case "getself":
		var resp admin.GetSelfResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		_ = table.Append([]string{style.label("Build name:"), resp.BuildName})
		_ = table.Append([]string{style.label("Build version:"), style.good(resp.BuildVersion)})
		_ = table.Append([]string{style.label("IPv6 address:"), resp.IPAddress})
		_ = table.Append([]string{style.label("IPv6 subnet:"), resp.Subnet})
		_ = table.Append([]string{style.label("Routing table size:"), fmt.Sprintf("%d", resp.RoutingEntries)})
		_ = table.Append([]string{style.label("Public key:"), resp.PublicKey})
		_ = table.Render()

	case "getpeers":
		var resp admin.GetPeersResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		table.Header(style.headers("URI", "State", "Dir", "IP Address", "Uptime", "RTT", "RX", "TX", "Down", "Up", "Pr", "Cost", "Last Error"))
		for _, peer := range resp.Peers {
			state, lasterr, dir, rtt, rxr, txr := "Up", "-", "Out", "-", "-", "-"
			if !peer.Up {
				if state = "Down"; peer.LastError != "" {
					lasterr = fmt.Sprintf("%s ago: %s", peer.LastErrorTime.Round(time.Second), peer.LastError)
				}
			} else if rttms := float64(peer.Latency.Microseconds()) / 1000; rttms > 0 {
				rtt = style.latency(fmt.Sprintf("%.02fms", rttms), rttms)
			}
			if peer.Inbound {
				dir = "In"
			}
			uristring := peer.URI
			if uri, err := url.Parse(peer.URI); err == nil {
				uri.RawQuery = ""
				uristring = uri.String()
			}
			if peer.RXRate > 0 {
				rxr = peer.RXRate.String() + "/s"
			}
			if peer.TXRate > 0 {
				txr = peer.TXRate.String() + "/s"
			}
			if lasterr != "-" {
				lasterr = style.bad(lasterr)
			}
			_ = table.Append([]string{
				uristring,
				style.status(state),
				dir,
				peer.IPAddress,
				(time.Duration(peer.Uptime) * time.Second).String(),
				rtt,
				peer.RXBytes.String(),
				peer.TXBytes.String(),
				rxr,
				txr,
				fmt.Sprintf("%d", peer.Priority),
				fmt.Sprintf("%d", peer.Cost),
				lasterr,
			})
		}
		_ = table.Render()

	case "gettree":
		var resp admin.GetTreeResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		table.Header(style.headers("Public Key", "IP Address", "Parent", "Sequence"))
		for _, tree := range resp.Tree {
			_ = table.Append([]string{
				tree.PublicKey,
				tree.IPAddress,
				tree.Parent,
				fmt.Sprintf("%d", tree.Sequence),
				//fmt.Sprintf("%d", dht.Port),
				//fmt.Sprintf("%d", dht.Rest),
			})
		}
		_ = table.Render()

	case "getpaths":
		var resp admin.GetPathsResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		table.Header(style.headers("Public Key", "IP Address", "Path", "Seq"))
		for _, p := range resp.Paths {
			_ = table.Append([]string{
				p.PublicKey,
				p.IPAddress,
				fmt.Sprintf("%v", p.Path),
				fmt.Sprintf("%d", p.Sequence),
			})
		}
		_ = table.Render()

	case "getsessions":
		var resp admin.GetSessionsResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		table.Header(style.headers("Public Key", "IP Address", "Uptime", "RX", "TX"))
		for _, p := range resp.Sessions {
			_ = table.Append([]string{
				p.PublicKey,
				p.IPAddress,
				(time.Duration(p.Uptime) * time.Second).String(),
				p.RXBytes.String(),
				p.TXBytes.String(),
			})
		}
		_ = table.Render()

	case "getnodeinfo":
		var resp core.GetNodeInfoResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		for _, v := range resp {
			fmt.Println(string(v))
			break
		}

	case "getmulticastinterfaces":
		var resp multicast.GetMulticastInterfacesResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		fmtBool := func(b bool) string {
			if b {
				return style.status("Yes")
			}
			return "-"
		}
		table.Header(style.headers("Name", "Listen Address", "Beacon", "Listen", "Password"))
		for _, p := range resp.Interfaces {
			_ = table.Append([]string{
				p.Name,
				p.Address,
				fmtBool(p.Beacon),
				fmtBool(p.Listen),
				fmtBool(p.Password),
			})
		}
		_ = table.Render()

	case "gettun":
		var resp tun.GetTUNResponse
		if err := json.Unmarshal(recv.Response, &resp); err != nil {
			panic(err)
		}
		enabled := fmt.Sprintf("%#v", resp.Enabled)
		if resp.Enabled {
			enabled = style.good(enabled)
		} else {
			enabled = style.warn(enabled)
		}
		_ = table.Append([]string{style.label("TUN enabled:"), enabled})
		if resp.Enabled {
			_ = table.Append([]string{style.label("Interface name:"), resp.Name})
			_ = table.Append([]string{style.label("Interface MTU:"), fmt.Sprintf("%d", resp.MTU)})
		}
		_ = table.Render()

	case "addpeer", "removepeer":

	default:
		fmt.Println(string(recv.Response))
	}

	return 0
}
