// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package net

import (
	"io/fs"
	"strings"
	"testing"
	"time"
)

type nssHostTest struct {
	host      string
	localhost string
	want      hostLookupOrder
}

func nssStr(s string) *nssConf { return parseNSSConf(strings.NewReader(s)) }

// represents a dnsConfig returned by parsing a nonexistent resolv.conf
var defaultResolvConf = &dnsConfig{
	servers:  defaultNS,
	ndots:    1,
	timeout:  5,
	attempts: 2,
	err:      fs.ErrNotExist,
}

func TestConfHostLookupOrder(t *testing.T) {
	tests := []struct {
		name      string
		c         *conf
		nss       *nssConf
		resolver  *Resolver
		hostTests []nssHostTest
	}{
		{
			name: "force",
			c: &conf{
				forceCgoLookupHost: true,
				resolv:             defaultResolvConf,
			},
			nss: nssStr("foo: bar"),
			hostTests: []nssHostTest{
				{"foo.local", "myhostname", hostLookupCgo},
				{"google.com", "myhostname", hostLookupCgo},
			},
		},
		{
			name: "netgo_dns_before_files",
			c: &conf{
				netGo:  true,
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: dns files"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupDNSFiles},
			},
		},
		{
			name: "netgo_fallback_on_cgo",
			c: &conf{
				netGo:  true,
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: dns files something_custom"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupFilesDNS},
			},
		},
		{
			name: "ubuntu_trusty_avahi",
			c: &conf{
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: files mdns4_minimal [NOTFOUND=return] dns mdns4"),
			hostTests: []nssHostTest{
				{"foo.local", "myhostname", hostLookupCgo},
				{"foo.local.", "myhostname", hostLookupCgo},
				{"foo.LOCAL", "myhostname", hostLookupCgo},
				{"foo.LOCAL.", "myhostname", hostLookupCgo},
				{"google.com", "myhostname", hostLookupFilesDNS},
			},
		},
		{
			name: "freebsdlinux_no_resolv_conf",
			c: &conf{
				goos:   "freebsd",
				resolv: defaultResolvConf,
			},
			nss:       nssStr("foo: bar"),
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupFilesDNS}},
		},
		// On OpenBSD, no resolv.conf means no DNS.
		{
			name: "openbsd_no_resolv_conf",
			c: &conf{
				goos:   "openbsd",
				resolv: defaultResolvConf,
			},
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupFiles}},
		},
		{
			name: "solaris_no_nsswitch",
			c: &conf{
				goos:   "solaris",
				resolv: defaultResolvConf,
			},
			nss:       &nssConf{err: fs.ErrNotExist},
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupCgo}},
		},
		{
			name: "openbsd_lookup_bind_file",
			c: &conf{
				goos:   "openbsd",
				resolv: &dnsConfig{lookup: []string{"bind", "file"}},
			},
			hostTests: []nssHostTest{
				{"google.com", "myhostname", hostLookupDNSFiles},
				{"foo.local", "myhostname", hostLookupDNSFiles},
			},
		},
		{
			name: "openbsd_lookup_file_bind",
			c: &conf{
				goos:   "openbsd",
				resolv: &dnsConfig{lookup: []string{"file", "bind"}},
			},
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupFilesDNS}},
		},
		{
			name: "openbsd_lookup_bind",
			c: &conf{
				goos:   "openbsd",
				resolv: &dnsConfig{lookup: []string{"bind"}},
			},
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupDNS}},
		},
		{
			name: "openbsd_lookup_file",
			c: &conf{
				goos:   "openbsd",
				resolv: &dnsConfig{lookup: []string{"file"}},
			},
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupFiles}},
		},
		{
			name: "openbsd_lookup_yp",
			c: &conf{
				goos:   "openbsd",
				resolv: &dnsConfig{lookup: []string{"file", "bind", "yp"}},
			},
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupCgo}},
		},
		{
			name: "openbsd_lookup_two",
			c: &conf{
				goos:   "openbsd",
				resolv: &dnsConfig{lookup: []string{"file", "foo"}},
			},
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupCgo}},
		},
		{
			name: "openbsd_lookup_empty",
			c: &conf{
				goos:   "openbsd",
				resolv: &dnsConfig{lookup: nil},
			},
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupDNSFiles}},
		},
		{
			name: "linux_no_nsswitch.conf",
			c: &conf{
				goos:   "linux",
				resolv: defaultResolvConf,
			},
			nss:       &nssConf{err: fs.ErrNotExist},
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupFilesDNS}},
		},
		{
			name: "linux_empty_nsswitch.conf",
			c: &conf{
				goos:   "linux",
				resolv: defaultResolvConf,
			},
			nss:       nssStr(""),
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupFilesDNS}},
		},
		{
			name: "files_mdns_dns",
			c: &conf{
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: files mdns dns"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupFilesDNS},
				{"x.local", "myhostname", hostLookupCgo},
			},
		},
		{
			name: "dns_special_hostnames",
			c: &conf{
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: dns"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupDNS},
				{"x\\.com", "myhostname", hostLookupCgo},     // punt on weird glibc escape
				{"foo.com%en0", "myhostname", hostLookupCgo}, // and IPv6 zones
			},
		},
		{
			name: "mdns_allow",
			c: &conf{
				resolv:       defaultResolvConf,
				hasMDNSAllow: true,
			},
			nss: nssStr("hosts: files mdns dns"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupCgo},
				{"x.local", "myhostname", hostLookupCgo},
			},
		},
		{
			name: "files_dns",
			c: &conf{
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: files dns"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupFilesDNS},
				{"x", "myhostname", hostLookupFilesDNS},
				{"x.local", "myhostname", hostLookupCgo},
			},
		},
		{
			name: "dns_files",
			c: &conf{
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: dns files"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupDNSFiles},
				{"x", "myhostname", hostLookupDNSFiles},
				{"x.local", "myhostname", hostLookupCgo},
			},
		},
		{
			name: "something_custom",
			c: &conf{
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: dns files something_custom"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupCgo},
			},
		},
		{
			name: "myhostname",
			c: &conf{
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: files dns myhostname"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupFilesDNS},
				{"myhostname", "myhostname", hostLookupCgo},
				{"myHostname", "myhostname", hostLookupCgo},
				{"myhostname.dot", "myhostname.dot", hostLookupCgo},
				{"myHostname.dot", "myhostname.dot", hostLookupCgo},
				{"_gateway", "myhostname", hostLookupCgo},
				{"_Gateway", "myhostname", hostLookupCgo},
				{"_outbound", "myhostname", hostLookupCgo},
				{"_Outbound", "myhostname", hostLookupCgo},
				{"localhost", "myhostname", hostLookupCgo},
				{"Localhost", "myhostname", hostLookupCgo},
				{"anything.localhost", "myhostname", hostLookupCgo},
				{"Anything.localhost", "myhostname", hostLookupCgo},
				{"localhost.localdomain", "myhostname", hostLookupCgo},
				{"Localhost.Localdomain", "myhostname", hostLookupCgo},
				{"anything.localhost.localdomain", "myhostname", hostLookupCgo},
				{"Anything.Localhost.Localdomain", "myhostname", hostLookupCgo},
				{"somehostname", "myhostname", hostLookupFilesDNS},
				{"", "myhostname", hostLookupFilesDNS}, // Issue 13623
			},
		},
		{
			name: "ubuntu14.04.02",
			c: &conf{
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: files myhostname mdns4_minimal [NOTFOUND=return] dns mdns4"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupFilesDNS},
				{"somehostname", "myhostname", hostLookupFilesDNS},
				{"myhostname", "myhostname", hostLookupCgo},
			},
		},
		// Debian Squeeze is just "dns,files", but lists all
		// the default criteria for dns, but then has a
		// non-standard but redundant notfound=return for the
		// files.
		{
			name: "debian_squeeze",
			c: &conf{
				resolv: defaultResolvConf,
			},
			nss: nssStr("hosts: dns [success=return notfound=continue unavail=continue tryagain=continue] files [notfound=return]"),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupDNSFiles},
				{"somehostname", "myhostname", hostLookupDNSFiles},
			},
		},
		{
			name: "resolv.conf-unknown",
			c: &conf{
				resolv: &dnsConfig{servers: defaultNS, ndots: 1, timeout: 5, attempts: 2, unknownOpt: true},
			},
			nss:       nssStr("foo: bar"),
			hostTests: []nssHostTest{{"google.com", "myhostname", hostLookupCgo}},
		},
		// Android should always use cgo.
		{
			name: "android",
			c: &conf{
				goos:   "android",
				resolv: defaultResolvConf,
			},
			nss: nssStr(""),
			hostTests: []nssHostTest{
				{"x.com", "myhostname", hostLookupCgo},
			},
		},
		// Issue 24393: make sure "Resolver.PreferGo = true" acts like netgo.
		{
			name:     "resolver-prefergo",
			resolver: &Resolver{PreferGo: true},
			c: &conf{
				goos:               "darwin",
				forceCgoLookupHost: true, // always true for darwin
				resolv:             defaultResolvConf,
				netCgo:             true,
			},
			nss: nssStr(""),
			hostTests: []nssHostTest{
				{"localhost", "myhostname", hostLookupFilesDNS},
			},
		},
	}

	origGetHostname := getHostname
	defer func() { getHostname = origGetHostname }()
	defer setSystemNSS(getSystemNSS(), 0)

	for _, tt := range tests {

		for _, ht := range tt.hostTests {
			getHostname = func() (string, error) { return ht.localhost, nil }
			setSystemNSS(tt.nss, time.Hour)

			gotOrder := tt.c.hostLookupOrder(tt.resolver, ht.host)
			if gotOrder != ht.want {
				t.Errorf("%s: hostLookupOrder(%q) = %v; want %v", tt.name, ht.host, gotOrder, ht.want)
			}
		}
	}
}

func setSystemNSS(nss *nssConf, addDur time.Duration) {
	nssConfig.mu.Lock()
	nssConfig.nssConf = nss
	nssConfig.mu.Unlock()
	nssConfig.acquireSema()
	nssConfig.lastChecked = time.Now().Add(addDur)
	nssConfig.releaseSema()
}

func TestSystemConf(t *testing.T) {
	systemConf()
}
