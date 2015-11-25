package nat

import (
	"bytes"
	"errors"
	"log"
	"net"
	"sync/atomic"
	"time"
)

// IsGlobalIP determs passed ip address is global or not.
// if ip address is global , it returns address type(ip4 or ip6).
func IsGlobalIP(trial net.IP) string {
	type localIPrange struct {
		from net.IP
		to   net.IP
	}
	locals := []localIPrange{
		localIPrange{net.ParseIP("10.0.0.0"), net.ParseIP("10.255.255.255")},
		localIPrange{net.ParseIP("172.16.0.0"), net.ParseIP("172.31.255.255")},
		localIPrange{net.ParseIP("192.168.0.0"), net.ParseIP("192.168.255.255")}}

	if trial == nil || trial.IsLoopback() {
		return ""
	}
	//for udp6
	if trial.To4() == nil {
		if trial.IsGlobalUnicast() {
			return "ip6"
		}
		return ""
	}
	//for udp4
	for _, r := range locals {
		if bytes.Compare(trial, r.from) >= 0 && bytes.Compare(trial, r.to) <= 0 {
			return ""
		}
	}
	return "ip4"
}

func externalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || IsGlobalIP(ip) == "" {
				continue
			}
			return ip, nil
		}
	}
	return nil, nil
}

//MappedPort holds mapped port information by NAT.
type MappedPort struct {
	Protocol     string
	InternalPort int
	ExternalPort *int32
	ch           chan string
}

//NetStatus holds NAT info, globlal IP address of this machine, and mapped port info by NAT.
type NetStatus struct {
	Nat        NAT
	GlobalIP   net.IP
	MappedPort []*MappedPort
}

//NewNetStatus make a new NetStatus struct, and gest global IP address from interface and NAT
//by UPnP and PMP.
func NewNetStatus() (*NetStatus, error) {
	var err error
	ns := NetStatus{}
	ns.MappedPort = make([]*MappedPort, 0)
	ns.GlobalIP, err = externalIP()
	if err != nil {
		return nil, err
	}
	if ns.GlobalIP != nil {
		return &ns, nil
	}
	ns.Nat, err = DiscoverGateway()
	if err != nil {
		return &ns, nil
	}
	ns.GlobalIP, err = ns.Nat.GetExternalAddress()
	if err != nil {
		return &ns, err
	}
	log.Println("found global IP address", ns.GlobalIP, "from NAT")
	return &ns, nil
}

//LoopPortMapping mapped port by NAT annd go looping to map continually.
func (ns *NetStatus) LoopPortMapping(protocol string, internalPort int, description string, timeout time.Duration) (*MappedPort, error) {
	if ns.Nat == nil {
		return nil, errors.New("nat is not found")
	}
	var err error
	m := MappedPort{}
	m.Protocol = protocol
	m.InternalPort = internalPort
	externalPort, err := ns.Nat.AddPortMapping(protocol, internalPort, description, timeout)
	if err != nil {
		return nil, err
	}
	log.Println("mapped", m.InternalPort, "to", externalPort)
	ep:=int32(externalPort)
	m.ExternalPort = &ep
	m.ch = make(chan string)
	ns.MappedPort = append(ns.MappedPort, &m)
	go func(n NAT, m *MappedPort, description string, timeout time.Duration) {
		for {
			select {
			case <-m.ch:
				return
			case <-time.Tick(time.Minute):
				ep, err := n.AddPortMapping(m.Protocol, m.InternalPort, description, timeout)
				if err != nil {
					log.Println("error: %s", err)
				}
				e:=int32(ep)
				atomic.StoreInt32(m.ExternalPort, e)

			}
		}
	}(ns.Nat, &m, description, timeout)
	return &m, nil
}

//StopPortMapping stops port mapping loop.
func (ns *NetStatus) StopPortMapping(m *MappedPort) error {
	if ns.Nat == nil {
		return errors.New("nat is not found")
	}
	m.ch <- "stop"
	log.Println("mapping is stopping")
	err := ns.Nat.DeletePortMapping(m.Protocol, m.InternalPort)
	log.Println("mapping is stopped")
	return err
}
