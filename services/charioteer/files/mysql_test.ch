// MySQL Test Script
setq(results, sqlQuery("mysql1", 'SELECT FirstName FROM `Person` WHERE `PersonID` = "09370e4d-3833-4e98-b4e4-dc6d5d3ad2cf"'))
setq(row, getAt(results, 0))
getProp(row, 'FirstName')`