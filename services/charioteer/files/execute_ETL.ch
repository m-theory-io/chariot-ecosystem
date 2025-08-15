// Execute ETL
doETL("fantasyisland", "Fan47K.csv", Fan47KtoSQL, map(
  "type", "sql",
  "connectionName", "mysql1",     // ← Uses mysql1 from bootstrap
  "tableName", "FantasyIsland"
))
