package etcdv3

import (
	"crypto/tls"
	"flag"
	"log"
	"strings"
	"time"

	_ "sparrow/session/providers/memory"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/pkg/transport"
	"google.golang.org/grpc"
)

var (
	// sep            = flag.String("sep", "/", "separator")
	// useAuth        = flag.Bool("auth", false, "use auth")
	// separator      = ""
	// sessmgr        *session.Manager
	// mu             sync.Mutex
	// rootUsers      = make(map[string]*userInfo) // host:rootUser
	usetls         = flag.Bool("usetls", false, "use tls")
	cacert         = flag.String("cacert", "", "verify certificates of TLS-enabled secure servers using this CA bundle (v3)")
	cert           = flag.String("cert", "", "identify secure client using this TLS certificate file (v3)")
	keyfile        = flag.String("key", "", "identify secure client using this TLS key file (v3)")
	connectTimeout = flag.Int("timeout", 5, "ETCD client connect timeout")
	epts           = flag.String("endpoints", "127.0.0.1:2379", "ETCD endpoints")
	endpoints      = []string{}
	Basepath       = flag.String("basepath", "/sparrow", "base path of etcd for storage")
)

// type userInfo struct {
// 	host   string
// 	uname  string
// 	passwd string
// }

func init() {
	endpoints = strings.Split(*epts, ",")
}

func NewClient() (*clientv3.Client, error) {
	// endpoints := []string{uinfo.host}
	var err error

	// use tls if usetls is true
	var tlsConfig *tls.Config
	if *usetls {
		tlsInfo := transport.TLSInfo{
			CertFile:      *cert,
			KeyFile:       *keyfile,
			TrustedCAFile: *cacert,
		}
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			log.Println(err.Error())
		}
	}

	conf := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Second * time.Duration(*connectTimeout),
		TLS:         tlsConfig,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	}

	var c *clientv3.Client
	c, err = clientv3.New(conf)
	if err != nil {
		return nil, err
	}
	return c, nil
}
