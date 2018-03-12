package tdns

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"time"

	"github.com/haxii/tdns/db/badger"
	"github.com/haxii/tdns/db/geoip"
	"github.com/haxii/tdns/proxy"
	"github.com/valyala/gorpc"
)

var (
	ipsTTL = time.Hour * 72
)

//NewRpcServer new tcp rpc server
func NewRpcServer(addr string) *gorpc.Server {
	d := gorpc.NewDispatcher()
	d.AddFunc("LookupIPAddr", LookupIPAddrs)
	d.AddFunc("SetProxyInfo", SetProxyInfo)
	d.AddFunc("ListProxyInfo", ListProxyInfo)
	return gorpc.NewTCPServer(addr, d.NewHandlerFunc())
}

type LookupIPRequest struct {
	IP   string
	Host string
}

type SetProxyRequest struct {
	Code string
	Addr string
	User string
	Pwd  string
}

//LookupIPAddrs rpc function
func LookupIPAddrs(req *LookupIPRequest) ([]net.IPAddr, error) {
	if len(req.IP) == 0 || len(req.Host) == 0 {
		return nil, errors.New("ip or host is empty")
	}
	countryCode, err := geoip.CountryCode(req.IP)
	if err != nil {
		return nil, err
	}
	//get ips from db
	ipskey := getHostIPsKey(req.Host, countryCode)
	bs, err := badger.Get(ipskey)
	if bs != nil {
		ips := make([]net.IPAddr, 1)
		err = json.Unmarshal(bs, &ips)
		if err != nil {
			return nil, err
		}
		return ips, err
	}

	// resove dns via proxy
	client := defaultProxyMng.GetProxyClient(countryCode)
	if client == nil {
		return nil, errors.New(fmt.Sprintf("not found socks for country(%s)", countryCode))
	}
	ips, err := client.ResoveDNS(req.Host, DNSAddr)
	if err != nil {
		return nil, err
	}

	//async save data to db
	go func() {
		defer func() {
			if r := recover(); r != nil {
				defaultLogger.Error("rpc", nil, "%v\n%s", r, debug.Stack())
			}
		}()

		bs, err := json.Marshal(ips)
		if err != nil {
			defaultLogger.Error("rpc", err, "", "")
		}
		err = badger.SetWithTTL(ipskey, bs, ipsTTL)
		if err != nil {
			defaultLogger.Error("rpc", err, "", "")
		}
	}()

	return ips, nil
}

//SetProxyInfo rpc function
func SetProxyInfo(req *SetProxyRequest) error {
	if len(req.Code) == 0 || len(req.Addr) == 0 {
		return errors.New("code or addr is empty")
	}
	err := defaultProxyMng.SetProxy(req.Code, req.Addr, req.User, req.Pwd)
	return err
}

//ListProxyInfo rpc function
func ListProxyInfo() (map[string]*proxy.ProxyInfo, error) {
	return defaultProxyMng.GetProxys()
}

func getHostIPsKey(host, countryCode string) []byte {
	return []byte(fmt.Sprintf("host:%s:%s", host, countryCode))
}
