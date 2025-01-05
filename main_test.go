package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNetworks(t *testing.T) {
	t.Log(IPv6())
	ns, err := Networks()
	if err != nil {
		t.Fatal(err)
	}
	en := json.NewEncoder(os.Stdout)
	en.SetIndent("", "  ")
	en.Encode(ns)

	t.Log(ns)
	t.Log(IsConnected(ns, "OpenWrt_5G"))
}
