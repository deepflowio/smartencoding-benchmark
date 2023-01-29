package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	_ "net/http/pprof"
	"os"
	"strings"
	"sync"
	"time"

	"cktest/ckdb"

	"cktest/ckwriter"

	"github.com/deepflowys/deepflow/server/libs/logger"
	"github.com/deepflowys/deepflow/server/libs/stats"
	logging "github.com/op/go-logging"
	yaml "gopkg.in/yaml.v2"
)

var log = logging.MustGetLogger("cktest")

var addr = flag.String("h", "127.0.0.1", "clickhouse ip")
var port = flag.Int("p", 9000, "clickhouse port")
var user = flag.String("u", "default", "clickhouse user")
var password = flag.String("passord", "", "clickhouse password")
var configFile = flag.String("f", "./config.yaml", "Specify config file location")

type ColumnField struct {
	Name            string `yaml:"name"`
	Type            string `yaml:"type"`
	Codec           string `yaml:"codec"`
	ValueRange      []int  `yaml:"value-range"` // [min,max,count]
	intValues       []int
	stringValues    []string
	IsNotOrderKey   bool `yaml:"is-not-order-key"`
	IsNotPrimaryKey bool `yaml:"is-not-primary-key"`
}

func (c *ColumnField) IsIntType() bool {
	return strings.Contains(c.Type, "Int")
}

func (c *ColumnField) IsStringType() bool {
	return strings.Contains(c.Type, "String")
}

type Config struct {
	DBName           string        `yaml:"db-name"`
	TableName        string        `yaml:"table-name"`
	DropTableIfExist bool          `yaml:"drop-table-if-exist"`
	Columns          []ColumnField `yaml:"columns,flow"`

	Engine  string `yaml:"engine"`
	Storage string `yaml:"storage"`
	TTL     int    `yaml:"ttl"`
	Cluster string `yaml:"cluster"`

	TimeInterval      int64 `yaml:"time-interval"`
	TimeIntervalWrite int   `yaml:"time-interval-write"`
	TimeOffset        int   `yaml:"time-offset"`

	WriteThreadCount int `yaml:"write-thread-count"`
	WriteTotalCount  int `yaml:"write-total-count"`
	WriteLoopCount   int `yaml:"write-loop-count"`
	BatchSize        int `yaml:"batch-size"`
	QueueSize        int `yaml:"queue-size"`
	SendRate         int `yaml:"send-rate"`

	StatEnabled   bool   `yaml:"stats-enabled"`
	StatsIP       string `yaml:"stats-ip"`
	StatsPort     uint   `yaml:"stats-port"`
	StatsInterval uint   `yaml:"stats-interval"`
}

func GetRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	for i := 0; i < l; i++ {
		result = append(result, bytes[randn(len(bytes))])
	}
	return string(result)
}

func randn(n int) int {
	if n == 0 {
		return 0
	}
	return rand.Intn(n)
}

func (c *Config) parseValues() error {
	for i := range c.Columns {
		col := &c.Columns[i]
		if len(col.ValueRange) != 3 {
			return fmt.Errorf("column name:%s value-range length must be 3([min,max,count])", col.Name)
		}
		min := col.ValueRange[0]
		max := col.ValueRange[1]
		count := col.ValueRange[2]
		if col.IsIntType() {
			if count > max-min {
				count = max - min
			}
			for i := 0; i < count; i++ {
				value := min + randn(max-min)
				col.intValues = append(col.intValues, value)
			}
		} else if col.IsStringType() {
			for i := 0; i < count; i++ {
				length := min
				if max > min {
					length = min + i%(max-min)
				}
				col.stringValues = append(col.stringValues, GetRandomString(length))
			}
		} else {
			return fmt.Errorf("column type is %s, unsupport column type not is int or string", col.Type)
		}
	}

	return nil
}

func (c *Config) genItem(time uint32, i int) writeItem {
	items := make(writeItem, 0, len(c.Columns)+1)
	items = append(items, time)
	var intItem int
	var strItem string
	for _, v := range c.Columns {
		min := v.ValueRange[0]
		max := v.ValueRange[1]
		if v.IsIntType() {
			if len(v.intValues) > 0 {
				intItem = v.intValues[i%len(v.intValues)]
			} else {
				intItem = min + randn(max-min)
			}
			var item interface{}
			switch v.Type {
			case "UInt64":
				item = uint32(intItem)
			case "UInt32":
				item = uint32(intItem)
			case "UInt16":
				item = uint16(intItem)
			case "UInt8":
				item = uint8(intItem)
			case "Int64":
				item = int32(intItem)
			case "Int32":
				item = int32(intItem)
			case "Int16":
				item = int16(intItem)
			case "Int8":
				item = int8(intItem)
			}
			items = append(items, item)
		} else {
			if len(v.stringValues) > 0 {
				strItem = v.stringValues[i%len(v.stringValues)]
			} else {
				length := min + randn(max-min)
				strItem = GetRandomString(length)
			}
			items = append(items, strItem)
		}
	}
	return items
}

func (c *Config) GenTable() *ckdb.Table {
	orderKeys := []string{}
	primaryKeyCount := 0
	columns := []*ckdb.Column{}
	columns = append(columns, ckdb.NewColumn("time", "DateTime('Asia/Shanghai')"))
	for _, v := range c.Columns {
		if !v.IsNotOrderKey {
			orderKeys = append(orderKeys, v.Name)
			if !v.IsNotPrimaryKey {
				primaryKeyCount += 1
			}
		}
		columns = append(columns, ckdb.NewColumn(v.Name, v.Type).SetIndex(ckdb.IndexNone.String()).SetCodec(ckdb.CodecDefault.String()))
	}
	return &ckdb.Table{
		ID:              0,
		Database:        c.DBName,
		LocalName:       c.TableName,
		GlobalName:      c.TableName + "_global",
		Columns:         columns,
		TimeKey:         "time",
		TTL:             c.TTL,
		PartitionFunc:   ckdb.TimeFuncHour,
		Cluster:         c.Cluster,
		StoragePolicy:   c.Storage,
		Engine:          c.Engine,
		OrderKeys:       orderKeys,
		PrimaryKeyCount: primaryKeyCount,
	}
}

func Load(path string) *Config {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Error("Read config file error:", err)
		os.Exit(1)
	}
	config := &Config{
		WriteLoopCount:   1,
		WriteThreadCount: 5,

		BatchSize:        100000,
		QueueSize:        1000000,
		TimeInterval:     60,
		SendRate:         1000000,
		DropTableIfExist: false,

		StatsIP:       "127.0.0.1",
		StatsPort:     20044,
		StatsInterval: 10,
		Engine:        "MergeTree()",
		TTL:           7, // day
		Storage:       "default",
	}

	if err = yaml.Unmarshal(configBytes, config); err != nil {
		log.Error("Unmarshal yaml error:", err)
		os.Exit(1)
	}
	fmt.Printf("%+v\n", *config)
	if err = config.parseValues(); err != nil {
		log.Error("Unmarshal yaml error:", err)
		os.Exit(1)
	}
	return config
}

type Writer struct {
	cfg      *Config
	Ckwriter *ckwriter.CKWriter
}

type writeItem []interface{}

func (i writeItem) Release() {
}

func (i writeItem) WriteBlock(block *ckdb.Block) {
	block.WriteDateTime(i[0].(uint32))
	for _, v := range i[1:] {
		block.Write(v)
	}
}

func NewWriter(cfg *Config) (*Writer, error) {
	writer, err := ckwriter.NewCKWriter([]string{fmt.Sprintf("%s:%d", *addr, *port)}, *user, *password,
		"ck-test", cfg.GenTable(), cfg.WriteThreadCount, cfg.QueueSize, cfg.BatchSize, 5)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	writer.Run()
	return &Writer{
		cfg:      cfg,
		Ckwriter: writer,
	}, nil
}

func sendWrite(threadID int, cfg *Config, writer *Writer) {
	rowTime := uint32(time.Now().Unix() + int64(cfg.TimeOffset))
	if cfg.TimeInterval > 0 {
		rowTime = rowTime / uint32(cfg.TimeInterval) * uint32(cfg.TimeInterval)
	}

	putCache := make(writeItem, 0, 1024)
	putCounter := 0
	putRate := cfg.SendRate / cfg.WriteThreadCount
	timeIntervalWrite := cfg.TimeIntervalWrite / cfg.WriteThreadCount
	beginTime := time.Now()
	for i := 0; i < cfg.WriteLoopCount; i++ {
		for j := threadID; j < cfg.WriteTotalCount; j += cfg.WriteThreadCount {
			putCache = append(putCache, cfg.genItem(rowTime, j))
			if len(putCache) >= 1024 {
				writer.Ckwriter.Put(putCache...)
				putCache = putCache[:0]
			}
			putCounter++
			if putCounter%timeIntervalWrite == 0 {
				rowTime += uint32(cfg.TimeInterval)
			}

			if putCounter%50000 == 0 {
				totalCostTime := time.Since(beginTime) / 1000000 // ms
				expectCostTime := putCounter * 1000 / putRate
				if totalCostTime < time.Duration(expectCostTime) {
					time.Sleep((time.Duration(expectCostTime) - totalCostTime) * time.Millisecond)
				}
			}
			if putCounter%2000000 == 0 {
				fmt.Printf("thread %d put writer count %d/%d cost time:%s\n", threadID, putCounter, cfg.WriteTotalCount/cfg.WriteThreadCount, time.Since(beginTime))
			}
		}
	}
	if len(putCache) >= 0 {
		writer.Ckwriter.Put(putCache...)
	}

	fmt.Printf("thread %d put writer count %d finish cost %s, wait 10s fow write \n", threadID, putCounter, time.Since(beginTime))
	time.Sleep(time.Second * 10)
	wg.Done()
}

var wg sync.WaitGroup

func main() {
	flag.Parse()

	logger.EnableStdoutLog()
	logger.EnableFileLog("./cktest.log")
	logging.SetLevel(4, "") // INFO

	cfg := Load(*configFile)
	if cfg.StatEnabled {
		stats.SetRemotes(fmt.Sprintf("%s:%d", cfg.StatsIP, int(cfg.StatsPort)))
		stats.SetMinInterval(time.Duration(cfg.StatsInterval) * time.Second)
	}

	writer, err := NewWriter(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("begin running: %s\n", time.Now())
	for i := 0; i < cfg.WriteThreadCount; i++ {
		wg.Add(1)
		go sendWrite(i, cfg, writer)
	}
	wg.Wait()
	fmt.Printf("end running: %s\n", time.Now())
}
