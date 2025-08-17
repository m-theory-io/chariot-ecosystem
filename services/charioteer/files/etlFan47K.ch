// etlFan47K

// Create transform object
declareGlobal(Fan47toCB, 'T', createTransform(Fan47toCB))
// Add mappings
addMapping(Fan47toCB, 'Sequence #', 'id', [], 'int', true)
addMapping(Fan47toCB, 'Market Area File Group', 'market_area_file_group', [], 'string', true)
addMapping(Fan47toCB, 'Hit', 'hit', [], 'string', true)
// return tree
Fan47toCB

