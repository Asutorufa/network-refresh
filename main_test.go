package main

import "testing"

func TestNetworks(t *testing.T) {
	t.Log(IPv6())
	ns, err := Networks()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ns)
	t.Log(IsConnected(ns, "OpenWrt_5G"))
}
