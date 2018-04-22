package g

import (
	"encoding/json"
	"log"
	"sync/atomic"
	"unsafe"

	"github.com/toolkits/file"
)

type File struct {
	Filename string
	Body     []byte
}

type HttpConfig struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
}

type RpcConfig struct {
	Enabled bool   `json:"enabled"`
	Listen  string `json:"listen"`
}

type RRDConfig struct {
	Storage string `json:"storage"`
}

type DBConfig struct {
	Dsn     string `json:"dsn"`
	MaxIdle int    `json:"maxIdle"`
}

type GlobalConfig struct {
	Pid         string      `json:"pid"`
	Debug       bool        `json:"debug"`
	Http        *HttpConfig `json:"http"`
	Rpc         *RpcConfig  `json:"rpc"`
	RRD         *RRDConfig  `json:"rrd"`
	DB          *DBConfig   `json:"db"`
	CallTimeout int32       `json:"callTimeout"`
	Migrate     struct {
		Concurrency int               `json:"concurrency"` //number of multiple worker per node
		Enabled     bool              `json:"enabled"`
		Replicas    int               `json:"replicas"`
		Cluster     map[string]string `json:"cluster"`
	} `json:"migrate"`
}

var (
	ConfigFile string
	ptr        unsafe.Pointer
)

/*
返回配置信息
 */
func Config() *GlobalConfig {
	// atomic.LoadPointer原子的读取
	return (*GlobalConfig)(atomic.LoadPointer(&ptr))
}

/*
加载配置文件到GlobalConfig ptr
 */
func ParseConfig(cfg string) {
	if cfg == "" {
		log.Fatalln("config file not specified: use -c $filename")
	}

	if !file.IsExist(cfg) {
		log.Fatalln("config file specified not found:", cfg)
	}

	ConfigFile = cfg

	configContent, err := file.ToTrimString(cfg)  // strings.TrimSpace(string(ioutil.ReadFile(cfg)))
	if err != nil {
		log.Fatalln("read config file", cfg, "error:", err.Error())
	}

	var c GlobalConfig
	err = json.Unmarshal([]byte(configContent), &c)  // string -> json
	if err != nil {
		log.Fatalln("parse config file", cfg, "error:", err.Error())
	}

	/*
	"migrate": {  //扩容graph时历史数据自动迁移
        "enabled": false,  //true or false, 表示graph是否处于数据迁移状态
        "concurrency": 2, //数据迁移时的并发连接数，建议保持默认
        "replicas": 500, //这是一致性hash算法需要的节点副本数量，建议不要变更，保持默认即可（必须和transfer的配置中保持一致）
        "cluster": { //未扩容前老的graph实例列表
            "graph-00" : "127.0.0.1:6070"
        }
    }
	 */
	if c.Migrate.Enabled && len(c.Migrate.Cluster) == 0 {
		c.Migrate.Enabled = false
	}

	// set config
	// unsafe.Pointer其实就是类似C的void *，在golang中是用于各种指针相互转换的桥梁
	// 详见https://my.oschina.net/goal/blog/193698
	// atomic.StorePointer用于原子的赋值，用在这里的意图是？？？？
	atomic.StorePointer(&ptr, unsafe.Pointer(&c))

	log.Println("g.ParseConfig ok, file", cfg)
}
