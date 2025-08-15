// Chariot test: CAS Preservation

// Set up Couchbase
if(not(exists('testcluster'))) {
    cbConnect('testcluster', '192.168.0.117', 'mtheory', 'Borg12731273')
    cbOpenBucket('testcluster', 'chariot')
}

// Insert document
log("DEBUG: creating testDoc variable")
setq(testDoc, parseJSONValue('{"type": "castest", "value": 123}'))
setq(result, cbInsert('testcluster', 'cas::test', testDoc))
// Remove document
cbRemove('testcluster', 'cas::test')

if(hasMeta(result, 'cas')) {
    // Verify CAS is string
    setq(cas, getMeta(result, 'cas'))
    setq(casType, typeOf(cas))
    equal(casType, 'S')
} else {
    "missing cas meta"
}
