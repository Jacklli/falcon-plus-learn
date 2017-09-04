package g

import (
	"encoding/json"
	"github.com/toolkits/file"
	"log"
	"strings"
	"sync"
)

type HttpConfig struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
}

type RpcConfig struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
}

type SocketConfig struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
	Timeout int    `json:"timeout"`
}

type JudgeConfig struct {
	Enabled     bool                    `json:"enabled"`
	Batch       int                     `json:"batch"`
	ConnTimeout int                     `json:"connTimeout"`
	CallTimeout int                     `json:"callTimeout"`
	MaxConns    int                     `json:"maxConns"`
	MaxIdle     int                     `json:"maxIdle"`
	Replicas    int                     `json:"replicas"`
	Cluster     map[string]string       `json:"cluster"`
	ClusterList map[string]*ClusterNode `json:"clusterList"`
}

type GraphConfig struct {
	Enabled     bool                    `json:"enabled"`
	Batch       int                     `json:"batch"`
	ConnTimeout int                     `json:"connTimeout"`
	CallTimeout int                     `json:"callTimeout"`
	MaxConns    int                     `json:"maxConns"`
	MaxIdle     int                     `json:"maxIdle"`
	Replicas    int                     `json:"replicas"`
	Cluster     map[string]string       `json:"cluster"`
	ClusterList map[string]*ClusterNode `json:"clusterList"`
}

type TsdbConfig struct {
	Enabled     bool   `json:"enabled"`
	Batch       int    `json:"batch"`
	ConnTimeout int    `json:"connTimeout"`
	CallTimeout int    `json:"callTimeout"`
	MaxConns    int    `json:"maxConns"`
	MaxIdle     int    `json:"maxIdle"`
	MaxRetry    int    `json:"retry"`
	Address     string `json:"address"`
}

type GlobalConfig struct {
	Debug   bool          `json:"debug"`
	MinStep int           `json:"minStep"` //最小周期,单位sec
	Http    *HttpConfig   `json:"http"`
	Rpc     *RpcConfig    `json:"rpc"`
	Socket  *SocketConfig `json:"socket"`
	Judge   *JudgeConfig  `json:"judge"`
	Graph   *GraphConfig  `json:"graph"`
	Tsdb    *TsdbConfig   `json:"tsdb"`
}

var (
	ConfigFile string
	config     *GlobalConfig
	configLock = new(sync.RWMutex)
)

func Config() *GlobalConfig {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

func ParseConfig(cfg string) {
	if cfg == "" {
		log.Fatalln("use -c to specify configuration file")
	}

	if !file.IsExist(cfg) {
		log.Fatalln("config file:", cfg, "is not existent. maybe you need `mv cfg.example.json cfg.json`")
	}

	ConfigFile = cfg

	configContent, err := file.ToTrimString(cfg)
	if err != nil {
		log.Fatalln("read config file:", cfg, "fail:", err)
	}

	var c GlobalConfig
	err = json.Unmarshal([]byte(configContent), &c)
	if err != nil {
		log.Fatalln("parse config file:", cfg, "fail:", err)
	}

	/*
	转换cluster配置的格式，map["node"]="host1,host2" --> map["node"]=["host1", "host2"]
	*/
	// split cluster config
	c.Judge.ClusterList = formatClusterItems(c.Judge.Cluster)
	c.Graph.ClusterList = formatClusterItems(c.Graph.Cluster)

	configLock.Lock()
	defer configLock.Unlock()
	config = &c

	log.Println("g.ParseConfig ok, file ", cfg)
}

// CLUSTER NODE
type ClusterNode struct {
	Addrs []string `json:"addrs"`
}

func NewClusterNode(addrs []string) *ClusterNode {
	return &ClusterNode{addrs}
}

/*
    "judge": {
        "enabled": true,
        "batch": 200,
        "connTimeout": 1000,
        "callTimeout": 5000,
        "maxConns": 32,
        "maxIdle": 32,
        "replicas": 500,
        "cluster": {
            "judge-00" : "host0:port0",
            "judge-01" : "host1:port1",
            "judge-02" : "host2:port2",
        }
    },
    "graph": {
        "enabled": true,
        "batch": 200,
        "connTimeout": 1000,
        "callTimeout": 5000,
        "maxConns": 32,
        "maxIdle": 32,
        "replicas": 500,
        "cluster": {
            "graph-00" : "host0a:port0a, host0b:port0b",
            "graph-01" : "host1a:port1a, host1b:port1b",
            "graph-02" : "host2a:port2a, host2b:port2b",
        }
    }
转换cluster配置的格式，map["node"]="host1,host2" --> map["node"]=["host1", "host2"]
        "cluster": {
            "judge-00" : "host0:port0",
            "judge-01" : "host1:port1",
            "judge-02" : "host2:port2",
        }
转换为
        "cluster": {
            "judge-00" : &ClusterNode{Addrs: ["host0:port0"],
            "judge-01" : &ClusterNode{Addrs: ["host1:port1"],
            "judge-02" : &ClusterNode{Addrs: ["host2:port2"],
        }

        "cluster": {
            "graph-00" : "host0a:port0a, host0b:port0b",
            "graph-01" : "host1a:port1a, host1b:port1b",
            "graph-02" : "host2a:port2a, host2b:port2b",
        }
转换为
        "cluster": {
            "graph-00" : &ClusterNode{Addrs: ["host0a:port0a", "host0b:port0b"],
            "graph-01" : &ClusterNode{Addrs: ["host1a:port1a", "host1b:port1b"],
            "graph-02" : &ClusterNode{Addrs: ["host2a:port2a", "host2b:port2b"],
        }
 */
// map["node"]="host1,host2" --> map["node"]=["host1", "host2"]
func formatClusterItems(cluster map[string]string) map[string]*ClusterNode {
	ret := make(map[string]*ClusterNode)
	for node, clusterStr := range cluster {
		items := strings.Split(clusterStr, ",")
		nitems := make([]string, 0)
		for _, item := range items {
			nitems = append(nitems, strings.TrimSpace(item))
		}
		ret[node] = NewClusterNode(nitems)
	}

	return ret
}
