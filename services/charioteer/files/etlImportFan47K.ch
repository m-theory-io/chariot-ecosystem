// etlImportFan47K
// Test generateCreateTable function
setq(csvFile, "Fan47K.csv")
setq(tableName, "FantasyIsland")
setq(sql, concat('USE testsql;\n', generateCreateTable(csvFile, tableName)))
sqlExecute(mysql1, sql)
