package main

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var (
	Iface = "wlan0"
)

func main() {
	timer := time.NewTicker(time.Hour * 2)

	StartCheck()
	for range timer.C {
		StartCheck()
	}
}

func StartCheck() {
	slog.Info("start check network")
	connected := true
	ns, err := Networks()
	if err == nil {
		connected = IsConnected(ns, "OpenWrt_5G")
	}

	if !connected || !TryIPv6() {
		Connect(ns, "OpenWrt_5G")
	}
	slog.Info("end check network")
}

var Dialer = &net.Dialer{
	ControlContext: func(ctx context.Context, network, address string, c syscall.RawConn) error {
		return c.Control(func(fd uintptr) {
			if err := unix.BindToDevice(int(fd), Iface); err != nil {
				slog.Error("Failed to bind to interface", "error", err)
				return
			}
		})
	},
}

var Client = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return Dialer.DialContext(ctx, network, addr)
		},
	},
	Timeout: time.Second * 15,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func TryIPv6() bool {
	for range 3 {
		ok := IPv6()
		if ok {
			return true
		}

		time.Sleep(time.Second * 5)
	}

	return false
}

func IPv6() bool {
	resp, err := Client.Get("http://[2a03:b0c0:3:d0::1a51:c001]")
	if err != nil {
		slog.Error("Failed to connect ipv6", "error", err)
		return false
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	slog.Info("Connected to ipv6", "status", resp.Status, "data", data)
	return true
}

func IsConnected(networks []Network, SSID string) bool {
	for _, network := range networks {
		if network.SSID == SSID {
			return network.Connected
		}
	}
	return false
}

type Network struct {
	Connected bool
	SSID      string
	BSSID     string
	MODE      string
	CHAN      string
	RATE      string
	SIGNAL    string
	BARS      string
	SECURITY  string
}

func Networks() ([]Network, error) {
	cmd := exec.Command("nmcli", "dev", "wifi", "list")

	data, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	scan := bufio.NewScanner(bytes.NewReader(data))

	scan.Scan()

	networks := make([]Network, 0)
	for scan.Scan() {
		fs := strings.Fields(scan.Text())

		if len(fs) < 9 {
			slog.Error("Invalid line", "line", scan.Text())
			continue
		}

		networks = append(networks, Network{
			Connected: fs[0] == "*",
			BSSID:     fs[1],
			SSID:      fs[2],
			MODE:      fs[3],
			CHAN:      fs[4],
			RATE:      fs[5],
			SIGNAL:    fs[6],
			BARS:      fs[7],
			SECURITY:  fs[8],
		})
	}

	return networks, nil
}

func Connect(ns []Network, network string) {
	has := slices.IndexFunc(ns, func(n Network) bool {
		return n.SSID == network
	})
	if has != -1 {
		slog.Error("can't find network", "network", network)
		return
	}

	cmd := exec.Command("nmcli", "dev", "wifi", "connect", network)
	data, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Failed to connect", "error", err, "data", data)
	}

	slog.Info("Connect result", "network", network, "data", data)
}
