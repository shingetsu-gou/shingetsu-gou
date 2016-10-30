package myself

import (
	"log"
	"net"
	"sync"
	"time"

	nat "github.com/shingetsu-gou/go-nat"
	"github.com/shingetsu-gou/shingetsu-gou/cfg"
)

var ip string
var externalPort *int32
var mutex sync.RWMutex
var status int

//init returns Myself obj.
func init() {
	resetConnection()
}

//resetConnectiontPort sets externalPort to internalPort.
func resetConnection() {
	mutex.Lock()
	defer mutex.Unlock()
	p := int32(cfg.DefaultPort)
	externalPort = &p
	status = cfg.Disconnected
}

//GetStatus returns status.
func GetStatus() int {
	mutex.RLock()
	defer mutex.RUnlock()
	return status
}

//SetStatus set connection status.
func SetStatus(stat int) {
	mutex.Lock()
	status = stat
	mutex.Unlock()
}

//GetIPPort returns ip address and external port number.
func GetIPPort() (string, int32) {
	mutex.RLock()
	defer mutex.RUnlock()
	return ip, *externalPort
}

//SetIP set my IP.
func SetIP(ips string) {
	mutex.Lock()
	defer mutex.Unlock()
	var nip net.IP
	if nip = net.ParseIP(ips); nip == nil {
		log.Println("ip", ips, "is illegal format")
		return
	}
	if nat.IsGlobalIP(nip) != "" {
		ip = ips
	}
}

//useUPnP gets external port by upnp and return external port.
//returns defaultPort if failed.
func useUPnP() bool {
	nt, err := nat.NewNetStatus()
	if err != nil {
		log.Println(err)
		return false
	}
	ma, err := nt.LoopPortMapping("tcp", cfg.DefaultPort, "shingetsu-gou", 10*time.Minute)
	if err != nil {
		log.Println(err)
		return false
	}
	externalPort = ma.ExternalPort
	return true
}

func connectionString() string {
	switch GetStatus() {
	case cfg.UPnP:
		return "uPnP"
	case cfg.Port0:
		return "failed"
	case cfg.Normal:
		return "normal"
	case cfg.Disconnected:
		return "disconnected"

	}
	return ""
}

//ResetPort setups connection.
func ResetPort() {
	if GetStatus() == cfg.Normal || GetStatus() == cfg.UPnP {
		return
	}
	switch cfg.NetworkMode {
	case cfg.Normal:
		resetConnection()
	case cfg.UPnP:
		if useUPnP() {
			SetStatus(cfg.UPnP)
		}
	}
	con := connectionString()
	log.Println("openned", con)
}
