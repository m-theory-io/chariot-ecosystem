// bootstrap.ch - loads startup runtime environment in dev server mode

// 1. connect to Couchbase
cbConnect('testCluster', '', '', '')
// 2. connect to RDBMS
sqlConnect('mysql1', '', '', '', '')
// 3. load usersAgent
declareGlobal(usersAgent, 'T', treeLoad('usersAgent.json'))
// 4. load ETL
declareGlobal(Fan47KtoSQL, 'T', treeLoad('Fan47KtoSQL.json'))
// 5. load BDI
declareGlobal(BDI, 'T', treeLoad('bdi_tree.json'))
// 6. agent start
agentStartNamed('thermostat', getAttribute(BDI, 'thermostat'))


