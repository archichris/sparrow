package cfg

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"sparrow/auth"
	"sparrow/etcdv3"
	"sparrow/session"
	_ "sparrow/session/providers/memory"

	"github.com/coreos/etcd/clientv3"
)

var (
	sep       = flag.String("sep", "/", "separator")
	useAuth   = flag.Bool("auth", false, "use auth")
	separator = ""
	sessmgr   *session.Manager
	mu        sync.Mutex
	CfgPath   = "cfg"
)

type userInfo struct {
	host   string
	uname  string
	passwd string
}

func init() {
	auth.HandleFunc("/v3/separator", GetSeparator)
	auth.HandleFunc("/v3/connect", Connect)
	auth.HandleFunc("/v3/put", Put)
	auth.HandleFunc("/v3/get", Get)
	auth.HandleFunc("/v3/delete", Del)
	auth.HandleFunc("/v3/getpath", GetPath)
	separator = *sep
}

func Test() {
	return
}

func nodesSort(node map[string]interface{}) {
	if v, ok := node["nodes"]; ok && v != nil {
		a := v.([]map[string]interface{})
		if len(a) != 0 {
			for i := 0; i < len(a)-1; i++ {
				// nodesSort(a[i])
				for j := i + 1; j < len(a); j++ {
					if a[j]["key"].(string) < a[i]["key"].(string) {
						a[i], a[j] = a[j], a[i]
					}
				}
			}
			for i := 0; i < len(a); i++ {
				nodesSort(a[i])
			}
		}
	}
}

func Connect(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	c, err := etcdv3.NewClient()
	if err != nil {
		log.Println(r.Method, "v3", "connect fail.")
		b, _ := json.Marshal(map[string]interface{}{"status": "error", "message": err.Error()})
		io.WriteString(w, string(b))
		return
	}
	defer c.Close()
	log.Println(r.Method, "v3", "connect success.")
	b, _ := json.Marshal(map[string]interface{}{"status": "ok"})
	io.WriteString(w, string(b))
}

func Put(w http.ResponseWriter, r *http.Request) {
	cli, _ := etcdv3.NewClient()
	defer cli.Close()
	key := r.FormValue("key")
	rKey := filepath.Join(*etcdv3.Basepath, CfgPath, key)
	value := r.FormValue("value")
	// ttl := r.FormValue("ttl")
	log.Println("PUT", "v3", rKey, key)

	var err error
	data := make(map[string]interface{})
	_, err = cli.Put(context.Background(), rKey, value)
	if err != nil {
		data["errorCode"] = 500
		data["message"] = err.Error()
	} else {
		if resp, err := cli.Get(context.Background(), rKey); err != nil {
			data["errorCode"] = 500
			data["errorCode"] = err.Error()
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				// node["key"] = string(kv.Key)
				node["key"] = key
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = getTTL(cli, kv.Lease)
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			}
		}
	}

	var dataByte []byte
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func Get(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]interface{})
	key := r.FormValue("key")
	rKey := filepath.Join(*etcdv3.Basepath, CfgPath, key)
	log.Println("GET", "v3", rKey, key)

	var cli *clientv3.Client
	cli, _ = etcdv3.NewClient()
	defer cli.Close()

	permissions := [][]string{{rKey, "p"}}

	if r.FormValue("prefix") == "true" {
		pnode := make(map[string]interface{})
		pnode["key"] = key
		pnode["nodes"] = make([]map[string]interface{}, 0)
		for _, p := range permissions {
			var (
				resp *clientv3.GetResponse
				err  error
			)
			if p[1] != "" {
				prefixKey := p[0]
				if p[0] == "/" {
					prefixKey = ""
				}
				resp, err = cli.Get(context.Background(), prefixKey, clientv3.WithPrefix())
			} else {
				resp, err = cli.Get(context.Background(), p[0])
			}
			if err != nil {
				data["errorCode"] = 500
				data["message"] = err.Error()
			} else {
				for _, kv := range resp.Kvs {
					node := make(map[string]interface{})
					node["key"] = strings.TrimPrefix(string(kv.Key), filepath.Join(*etcdv3.Basepath, CfgPath))
					node["value"] = string(kv.Value)
					node["dir"] = false
					if rKey == string(kv.Key) {
						node["ttl"] = getTTL(cli, kv.Lease)
					} else {
						node["ttl"] = 0
					}
					node["createdIndex"] = kv.CreateRevision
					node["modifiedIndex"] = kv.ModRevision
					nodes := pnode["nodes"].([]map[string]interface{})
					pnode["nodes"] = append(nodes, node)
				}
			}
		}
		data["node"] = pnode
	} else {
		if resp, err := cli.Get(context.Background(), rKey); err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
		} else {
			if resp.Count > 0 {
				kv := resp.Kvs[0]
				node := make(map[string]interface{})
				node["key"] = strings.TrimPrefix(string(kv.Key), filepath.Join(*etcdv3.Basepath, CfgPath))
				node["value"] = string(kv.Value)
				node["dir"] = false
				node["ttl"] = getTTL(cli, kv.Lease)
				node["createdIndex"] = kv.CreateRevision
				node["modifiedIndex"] = kv.ModRevision
				data["node"] = node
			} else {
				data["errorCode"] = 500
				data["message"] = "The node does not exist."
			}
		}
	}

	var dataByte []byte
	var err error
	if dataByte, err = json.Marshal(data); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func GetPath(w http.ResponseWriter, r *http.Request) {
	key := r.FormValue("key")
	cfgPath := filepath.Join(*etcdv3.Basepath, CfgPath)
	rKey := filepath.Join(cfgPath, key)

	log.Println("GET", "v3", rKey, key)
	var (
		data = make(map[string]interface{})
		/*
			{1:["/"], 2:["/foo", "/foo2"], 3:["/foo/bar", "/foo2/bar"], 4:["/foo/bar/test"]}
		*/
		all = make(map[int][]map[string]interface{})
		min int
		max int
		//prefixKey string
	)

	var cli *clientv3.Client
	cli, _ = etcdv3.NewClient()
	defer cli.Close()

	permissions := [][]string{{rKey, "p"}}

	var (
		presp *clientv3.GetResponse
		err   error
	)
	if key != separator {
		presp, err = cli.Get(context.Background(), rKey)
		if err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
			dataByte, _ := json.Marshal(data)
			io.WriteString(w, string(dataByte))
			return
		}
	}
	if key == separator {
		min = 1
		//prefixKey = separator
	} else {
		min = len(strings.Split(key, separator))
		//prefixKey = originKey
	}
	max = min
	all[min] = []map[string]interface{}{{"key": key}}
	if presp != nil && presp.Count != 0 {
		all[min][0]["value"] = string(presp.Kvs[0].Value)
		all[min][0]["ttl"] = getTTL(cli, presp.Kvs[0].Lease)
		all[min][0]["createdIndex"] = presp.Kvs[0].CreateRevision
		all[min][0]["modifiedIndex"] = presp.Kvs[0].ModRevision
	}
	all[min][0]["nodes"] = make([]map[string]interface{}, 0)

	for _, p := range permissions {
		key, rangeEnd := p[0], p[1]
		//child
		var resp *clientv3.GetResponse
		if rangeEnd != "" {
			resp, err = cli.Get(context.Background(), key, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
		} else {
			resp, err = cli.Get(context.Background(), key, clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend))
		}
		if err != nil {
			data["errorCode"] = 500
			data["message"] = err.Error()
			dataByte, _ := json.Marshal(data)
			io.WriteString(w, string(dataByte))
			return
		}

		for _, kv := range resp.Kvs {
			if string(kv.Key) == cfgPath {
				continue
			}
			keys := strings.Split(strings.TrimPrefix(string(kv.Key), cfgPath), separator) // /foo/bar
			for i := range keys {                                                         // ["", "foo", "bar"]
				k := strings.Join(keys[0:i+1], separator)
				if k == "" {
					continue
				}
				// k = filepath.Join(cfgPath, k)
				node := map[string]interface{}{"key": k}
				if node["key"].(string) == strings.TrimPrefix(string(kv.Key), cfgPath) {
					node["value"] = string(kv.Value)
					if key == string(kv.Key) {
						node["ttl"] = getTTL(cli, kv.Lease)
					} else {
						node["ttl"] = 0
					}
					node["createdIndex"] = kv.CreateRevision
					node["modifiedIndex"] = kv.ModRevision
				}
				level := len(strings.Split(k, separator))
				if level > max {
					max = level
				}

				if _, ok := all[level]; !ok {
					all[level] = make([]map[string]interface{}, 0)
				}
				levelNodes := all[level]
				var isExist bool
				for _, n := range levelNodes {
					if n["key"].(string) == k {
						isExist = true
					}
				}
				if !isExist {
					node["nodes"] = make([]map[string]interface{}, 0)
					all[level] = append(all[level], node)
				}
			}
		}
	}

	// parent-child mapping
	for i := max; i > min; i-- {
		for _, a := range all[i] {
			for _, pa := range all[i-1] {
				if i == 2 {
					pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
					pa["dir"] = true
				} else {
					if strings.HasPrefix(a["key"].(string), pa["key"].(string)+separator) {
						pa["nodes"] = append(pa["nodes"].([]map[string]interface{}), a)
						pa["dir"] = true
					}
				}
			}
		}
	}
	// }
	data = all[min][0]
	if dataByte, err := json.Marshal(map[string]interface{}{"node": data}); err != nil {
		io.WriteString(w, err.Error())
	} else {
		io.WriteString(w, string(dataByte))
	}
}

func Del(w http.ResponseWriter, r *http.Request) {
	cli, _ := etcdv3.NewClient()
	defer cli.Close()
	key := r.FormValue("key")
	rKey := filepath.Join(*etcdv3.Basepath, CfgPath, key)
	dir := r.FormValue("dir")
	log.Println("DELETE", "v3", rKey, key)

	if _, err := cli.Delete(context.Background(), rKey); err != nil {
		io.WriteString(w, err.Error())
		return
	}

	if dir == "true" {
		if _, err := cli.Delete(context.Background(), rKey+separator, clientv3.WithPrefix()); err != nil {
			io.WriteString(w, err.Error())
			return
		}
	}
	io.WriteString(w, "ok")
}

func getTTL(cli *clientv3.Client, lease int64) int64 {
	resp, err := cli.Lease.TimeToLive(context.Background(), clientv3.LeaseID(lease))
	if err != nil {
		return 0
	}
	if resp.TTL == -1 {
		return 0
	}
	return resp.TTL
}

func GetSeparator(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, separator)
}


func getInfo(host string) map[string]string {
	info := make(map[string]string)
	// uinfo := rootUsers[host]
	rootClient, err := etcdv3.NewClient()
	if err != nil {
		log.Println(err)
		return info
	}
	defer rootClient.Close()

	status, err := rootClient.Status(context.Background(), host)
	if err != nil {
		log.Fatal(err)
	}
	mems, err := rootClient.MemberList(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	kb := 1024
	mb := kb * 1024
	gb := mb * 1024
	var sizeStr string
	for _, m := range mems.Members {
		if m.ID == status.Leader {
			info["version"] = status.Version
			gn, rem1 := size(int(status.DbSize), gb)
			mn, rem2 := size(rem1, mb)
			kn, bn := size(rem2, kb)
			if sizeStr != "" {
				sizeStr += " "
			}
			if gn > 0 {
				info["size"] = fmt.Sprintf("%dG", gn)
			} else {
				if mn > 0 {
					info["size"] = fmt.Sprintf("%dM", mn)
				} else {
					if kn > 0 {
						info["size"] = fmt.Sprintf("%dK", kn)
					} else {
						info["size"] = fmt.Sprintf("%dByte", bn)
					}
				}
			}
			info["name"] = m.GetName()
			break
		}
	}
	return info
}

func size(num int, unit int) (n, rem int) {
	return num / unit, num - (num/unit)*unit
}
