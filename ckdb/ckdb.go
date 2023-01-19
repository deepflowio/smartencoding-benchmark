package ckdb

import (
	"fmt"
	"net"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	logging "github.com/op/go-logging"
)

var log = logging.MustGetLogger("ckdb")

const DEFAULT_COLUMN_COUNT = 256

type Block struct {
	batch driver.Batch
	items []interface{}
}

func NewBlock(batch driver.Batch) *Block {
	return &Block{
		batch: batch,
		items: make([]interface{}, 0, DEFAULT_COLUMN_COUNT),
	}
}

func (b *Block) WriteAll() error {
	err := b.batch.Append(b.items...)
	b.items = b.items[:0]
	return err
}

func (b *Block) Send() error {
	return b.batch.Send()
}

func (b *Block) Write(v ...interface{}) {
	b.items = append(b.items, v...)
}

func (b *Block) WriteDateTime(v uint32) {
	b.items = append(b.items, time.Unix(int64(v), 0))
}

func (b *Block) WriteIPv6(v net.IP) {
	if len(v) == 0 {
		v = net.IPv6zero
	}
	b.items = append(b.items, v)
}

type ColumnType uint8

const (
	UInt64 ColumnType = iota
	UInt64Nullable
	UInt32
	UInt32Nullable
	UInt16
	UInt16Nullable
	UInt8
	UInt8Nullable
	Int64
	Int64Nullable
	Int32
	Int32Nullable
	Int16
	Int16Nullable
	Int8
	Int8Nullable
	Float64
	Float64Nullable
	String
	IPv6
	IPv4
	ArrayString
	ArrayUInt8
	ArrayUInt16
	ArrayUInt32
	ArrayInt64
	ArrayFloat64
	DateTime
	DateTime64
	DateTime64ms
	DateTime64us
	FixString8
	LowCardinalityString
)

var cloumnTypeString = []string{
	UInt64:               "UInt64",
	UInt64Nullable:       "Nullable(UInt64)",
	UInt32:               "UInt32",
	UInt32Nullable:       "Nullable(UInt32)",
	UInt16:               "UInt16",
	UInt16Nullable:       "Nullable(UInt16)",
	UInt8:                "UInt8",
	UInt8Nullable:        "Nullable(UInt8)",
	Int64:                "Int64",
	Int64Nullable:        "Nullable(Int64)",
	Int32:                "Int32",
	Int32Nullable:        "Nullable(Int32)",
	Int16:                "Int16",
	Int16Nullable:        "Nullable(Int16)",
	Int8:                 "Int8",
	Int8Nullable:         "Nullable(Int8)",
	Float64:              "Float64",
	Float64Nullable:      "Nullable(Float64)",
	String:               "String",
	IPv6:                 "IPv6",
	IPv4:                 "IPv4",
	ArrayString:          "Array(String)",
	ArrayUInt8:           "Array(UInt8)",
	ArrayUInt16:          "Array(UInt16)",
	ArrayUInt32:          "Array(UInt32)",
	ArrayInt64:           "Array(Int64)",
	ArrayFloat64:         "Array(Float64)",
	DateTime:             "DateTime('Asia/Shanghai')",
	DateTime64:           "DateTime64(0, 'Asia/Shanghai')",
	DateTime64ms:         "DateTime64(3, 'Asia/Shanghai')",
	DateTime64us:         "DateTime64(6, 'Asia/Shanghai')",
	FixString8:           "FixedString(8)",
	LowCardinalityString: "LowCardinality(String)",
}

func (t ColumnType) String() string {
	return cloumnTypeString[t]
}

type CodecType uint8

const (
	CodecDefault CodecType = iota // lz4
	CodecLZ4
	CodecLZ4HC
	CodecZSTD
	CodecT64
	CodecDelta
	CodecDoubleDelta
	CodecGorilla
	CodecNone
)

var codecTypeString = []string{
	CodecDefault:     "",
	CodecLZ4:         "LZ4",
	CodecLZ4HC:       "LZ4HC",
	CodecZSTD:        "ZSTD",
	CodecT64:         "T64",
	CodecDelta:       "Delta",
	CodecDoubleDelta: "DoubleDelta",
	CodecGorilla:     "Gorilla",
	CodecNone:        "None",
}

func (t CodecType) String() string {
	return codecTypeString[t]
}

type IndexType uint8

const (
	IndexNone IndexType = iota
	IndexMinmax
	IndexSet
	IndexBloomfilter
)

var indexTypeString = []string{
	IndexNone:        "",
	IndexMinmax:      "minmax",
	IndexSet:         "set(300)",
	IndexBloomfilter: "bloom_filter",
}

func (t IndexType) String() string {
	return indexTypeString[t]
}

type TimeFuncType uint8

const (
	TimeFuncNone TimeFuncType = iota
	TimeFuncMinute
	TimeFuncTenMinute
	TimeFuncHour
	TimeFuncFourHour
	TimeFuncTwelveHour
	TimeFuncDay
	TimeFuncWeek
	TimeFuncMonth
	TimeFuncYYYYMM
	TimeFuncYYYYMMDD
)

var timeFuncTypeString = []string{
	TimeFuncNone:       "%s",
	TimeFuncMinute:     "toStartOfMinute(%s)", // %s指代函数作用于的字段名
	TimeFuncTenMinute:  "toStartOfTenMinute(%s)",
	TimeFuncHour:       "toStartOfHour(%s)",
	TimeFuncFourHour:   "toStartOfInterval(%s, INTERVAL 4 hour)",
	TimeFuncTwelveHour: "toStartOfInterval(%s, INTERVAL 12 hour)",
	TimeFuncDay:        "toStartOfDay(%s)",
	TimeFuncWeek:       "toStartOfWeek(%s)",
	TimeFuncMonth:      "toStartOfMonth(%s)",
	TimeFuncYYYYMM:     "toYYYYMM(%s)",
	TimeFuncYYYYMMDD:   "toYYYYMMDD(%s)",
}

func (t TimeFuncType) String(timeKey string) string {
	return fmt.Sprintf(timeFuncTypeString[t], timeKey)
}

type EngineType uint8

const (
	Distributed EngineType = iota
	MergeTree
	ReplicatedMergeTree
	AggregatingMergeTree
	ReplicatedAggregatingMergeTree
	ReplacingMergeTree
	SummingMergeTree
)

var engineTypeString = []string{
	Distributed:                    "Distributed('%s', '%s', '%s', rand())", // %s %s %s 指代cluster名，数据库名，表名
	MergeTree:                      "MergeTree()",
	ReplicatedMergeTree:            "ReplicatedMergeTree('/clickhouse/tables/{shard}/%s/%s', '{replica}')", // 字符串参数表示zk的路径，shard和replica自动从clickhouse的macros读取, %s/%s分别指代数据库名和表名
	AggregatingMergeTree:           "AggregatingMergeTree()",
	ReplicatedAggregatingMergeTree: "ReplicatedAggregatingMergeTree('/clickhouse/tables/{shard}/%s/%s', '{replica}')",
	ReplacingMergeTree:             "ReplacingMergeTree(%s)",
	SummingMergeTree:               "SummingMergeTree(%s)",
}

func (t EngineType) String() string {
	return engineTypeString[t]
}

type DiskType uint8

const (
	Volume DiskType = iota
	Disk
)

func (t DiskType) String() string {
	switch t {
	case Volume:
		return "VOLUME"
	case Disk:
		return "DISK"
	}
	return "Unknown"
}

const (
	DF_STORAGE_POLICY     = "df_storage"
	DF_CLUSTER            = "df_cluster"
	DF_REPLICATED_CLUSTER = "df_replicated_cluster"
)

type Column struct {
	Name    string // 列名
	Type    string // 数据类型
	Codec   string // 压缩算法
	Index   string // 二级索引
	GroupBy bool   // 在AggregatingMergeTree表中用于group by的字段
	Comment string // 列注释
}

func (c *Column) SetGroupBy() *Column {
	c.GroupBy = true
	return c
}

func (c *Column) SetCodec(ct string) *Column {
	c.Codec = ct
	return c
}

func (c *Column) SetIndex(i string) *Column {
	c.Index = i
	return c
}

func (c *Column) SetComment(comment string) *Column {
	c.Comment = comment
	return c
}

func NewColumn(name string, t string) *Column {
	index := IndexNone.String()
	codec := CodecDefault.String()
	switch t {
	case UInt8.String(): // u8默认设置set的二级索引
		index = IndexSet.String()
	case UInt16.String(), UInt32.String(), Int32.String(), IPv4.String(), IPv6.String(), ArrayUInt16.String(): // 默认设置minmax的二级索引
		index = IndexMinmax.String()
	case UInt64.String(), Int64.String():
		codec = CodecT64.String()
	case DateTime.String(), DateTime64ms.String(), DateTime64us.String():
		codec = CodecDoubleDelta.String()
		index = IndexMinmax.String() // 时间默认设置minmax的二级索引
	case Float64.String():
		codec = CodecGorilla.String()
	}
	return &Column{name, t, codec, index, false, ""}
}

func NewColumnWithGroupBy(name string, t string) *Column {
	return NewColumn(name, t).SetGroupBy()
}

func NewColumns(names []string, t string) []*Column {
	columns := make([]*Column, 0, len(names))
	for _, name := range names {
		columns = append(columns, NewColumn(name, t))
	}
	return columns
}

// nameComments: 需要同时创建的列列表，列表元素是长度为2的字符串数组, 第一个元素是列名，第二个是注释内容
func NewColumnsWithComment(nameComments [][2]string, t string) []*Column {
	columns := make([]*Column, 0, len(nameComments))
	for _, nameComment := range nameComments {
		columns = append(columns, NewColumn(nameComment[0], t).SetComment(nameComment[1]))
	}
	return columns
}
