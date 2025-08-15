// etlMakeMaps
setq(table_name, 'FantasyIsland')
setq(query, 'SELECT column_name, ordinal_position, is_nullable, data_type, character_maximum_length, numeric_precision, numeric_scale, column_type, column_key FROM information_schema.columns WHERE table_name = "${table_name}";')
setq(qres, sqlQuery('mysql1', query))
// Get headers
setq(headers, 'Fan47K.csv')
// make maps
setq(counter, 0)
setq(numCols, length(qres))
setq(numFields, length(headers))
setq(offset, sub(numCols, numFields))
while(smaller(counter, numFields)) {
    setq(field, getAt(headers, counter))
    setq(column, getAt(qres, add(counter, offset)))
    // ("addMapping requires at least 6 arguments: transform, sourceField, targetColumn, program, dataType, required"
    addMapping(Fan47KtoCB, field, getAttribute(column, 'COLUMN_NAME'), [], getAttribute(column, 'COLUMN_TYPE'), func(column) {
        if(equal(getAttribute(column, 'IS_NULLABLE'), 'NO')) {
            true
        } else {
            false
        }
    })
    setq(counter, add(counter, 1))
}
Fan47KtoCB