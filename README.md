# smartencoding-benchmark

## 编译
 - make

## 运行
 - 配置config.yaml
 - ./cktest -h <clickhouse ip> -p <clickhouse port> 

## 查看结果
 - 通过查询`system.parts`信息，可以获取表占用磁盘空间等。`select * from system.parts where database='test'`
 - 通过查询`system.parts_columns`信息，可以获取各列占用磁盘空间等
  ```
SELECT
    column,
    any(type),
    formatReadableSize(sum(column_data_uncompressed_bytes)) AS `原始大小`,
    formatReadableSize(sum(column_data_compressed_bytes)) AS `压缩大小`,
    sum(rows)
FROM system.parts_columns
WHERE (database = 'test') AND (table = 'test')
GROUP BY column
ORDER BY column ASC

  ```
## 工具说明
  1. 配置需要写入的表结构
      - 列名，类型，每列数据的取值范围等
  2. 根据配置预先生成每列数据的取值范围数据池
  3. 多线程实时从数据池中获取数据，并生成1行数据，发送到多个写入队列中(默认1个队列可以缓存100w条数据)
  4. 多线程从写入队列读取数据(默认每次10w条),执行一次写入CK
      - 可以根据配置的写入线程数和写入速率，控制写入速率
