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

