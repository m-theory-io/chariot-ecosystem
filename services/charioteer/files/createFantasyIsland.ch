// etlImportFan47K
// Test generateCreateTable function
setq(csvFile, "Fan47K.csv")
setq(tableName, "FantasyIsland")
setq(sql, concat('USE testsql;', generateCreateTable(csvFile, tableName)))
sqlExecute('mysql1', sql)
