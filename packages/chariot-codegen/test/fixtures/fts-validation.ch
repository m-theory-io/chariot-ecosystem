// Generated from FTS modelseed. Use create<Class>(attributes, state) for checked construction.
class(Sample) {
field('count', 'V', null)
field('empty', 'V', null)
field('id', 'S')
field('label', 'V', null)
field('mode', 'V', null)
field('nothing', 'V', null)
field('settings', 'V', null)
field('tags', 'V', null)
field('ftsState', 'M', mapValue('status', 'active'))
field('ftsExtraAttributes', 'M', mapValue())
}
setq(validateSample, withSignature(func(attributes, state) {
setq(attrs, mapValue())
setq(stateValues, mapValue('status', 'active'))
apply(func(key, value) {
setProp(attrs, key, value)
}, attributes)
apply(func(key, value) {
setProp(stateValues, key, value)
}, state)
if (not(or(equal(valueType(attrs), 'M')))) {
error('FTS Sample.attributes: expected object')
}
if (equal(valueType(attrs), 'M')) {
setq(seen0, mapValue())
apply(func(key0, item0) {
setProp(seen0, key0, true)
if (equal(key0, 'count')) {
if (not(or(equal(valueType(item0), 'N'), equal(item0, null)))) {
error('FTS Sample.attributes.count: expected integer|null')
}
if (equal(valueType(item0), 'N')) {
if (not(equal(floor(item0), item0))) {
error('FTS Sample.attributes.count: expected integer')
}
if (not(biggerEq(item0, 0))) {
error('FTS Sample.attributes.count: minimum')
}
if (not(smallerEq(item0, 10))) {
error('FTS Sample.attributes.count: maximum')
}
}
}
if (equal(key0, 'empty')) {
if (not(or(equal(valueType(item0), 'M')))) {
error('FTS Sample.attributes.empty: expected object')
}
if (equal(valueType(item0), 'M')) {
setq(seen1, mapValue())
apply(func(key1, item1) {
setProp(seen1, key1, true)
if (not(false)) {
error('FTS Sample.attributes.empty: unknown field')
}
}, item0)
}
}
if (equal(key0, 'id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Sample.attributes.id: expected string')
}
}
if (equal(key0, 'label')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Sample.attributes.label: expected string')
}
if (equal(valueType(item0), 'S')) {
if (not(biggerEq(length(split(item0, '')), 1))) {
error('FTS Sample.attributes.label: minLength')
}
if (not(smallerEq(length(split(item0, '')), 2))) {
error('FTS Sample.attributes.label: maxLength')
}
}
}
if (equal(key0, 'mode')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Sample.attributes.mode: expected string')
}
if (not(or(equal(item0, 'a'), equal(item0, 'b')))) {
error('FTS Sample.attributes.mode: not in enum')
}
}
if (equal(key0, 'nothing')) {
if (not(or(equal(item0, null)))) {
error('FTS Sample.attributes.nothing: expected null')
}
}
if (equal(key0, 'settings')) {
if (not(or(equal(valueType(item0), 'M')))) {
error('FTS Sample.attributes.settings: expected object')
}
if (equal(valueType(item0), 'M')) {
setq(seen2, mapValue())
apply(func(key2, item2) {
setProp(seen2, key2, true)
if (equal(key2, 'enabled')) {
if (not(or(equal(valueType(item2), 'L')))) {
error('FTS Sample.attributes.settings.enabled: expected boolean')
}
}
if (not(or(equal(key2, 'enabled')))) {
error('FTS Sample.attributes.settings: unknown field')
}
}, item0)
if (not(equal(getProp(seen2, 'enabled'), true))) {
error('FTS Sample.attributes.settings.enabled: required field')
}
}
}
if (equal(key0, 'tags')) {
if (not(or(equal(valueType(item0), 'A')))) {
error('FTS Sample.attributes.tags: expected array')
}
if (equal(valueType(item0), 'A')) {
if (not(biggerEq(length(item0), 1))) {
error('FTS Sample.attributes.tags: minItems')
}
if (not(smallerEq(length(item0), 2))) {
error('FTS Sample.attributes.tags: maxItems')
}
}
if (equal(valueType(item0), 'A')) {
apply(func(index3, item3) {
if (not(or(equal(valueType(item3), 'S')))) {
error('FTS Sample.attributes.tags[]: expected string')
}
}, item0)
}
}
if (not(or(equal(key0, 'count'), equal(key0, 'empty'), equal(key0, 'id'), equal(key0, 'label'), equal(key0, 'mode'), equal(key0, 'nothing'), equal(key0, 'settings'), equal(key0, 'tags')))) {
error('FTS Sample.attributes: unknown field')
}
}, attrs)
if (not(equal(getProp(seen0, 'id'), true))) {
error('FTS Sample.attributes.id: required field')
}
}
if (not(or(equal(valueType(stateValues), 'M')))) {
error('FTS Sample.state: expected object')
}
if (equal(valueType(stateValues), 'M')) {
setq(seen10000, mapValue())
apply(func(key10000, item10000) {
setProp(seen10000, key10000, true)
if (equal(key10000, 'status')) {
if (not(or(equal(valueType(item10000), 'S')))) {
error('FTS Sample.state.status: expected string')
}
if (not(or(equal(item10000, 'active'), equal(item10000, 'inactive')))) {
error('FTS Sample.state.status: not in enum')
}
}
if (not(or(equal(key10000, 'status')))) {
error('FTS Sample.state: unknown field')
}
}, stateValues)
if (not(equal(getProp(seen10000, 'status'), true))) {
error('FTS Sample.state.status: required field')
}
}
true
}, signature(array('M', 'M'), 'L')))
setq(createSample, withSignature(func(attributes, state) {
validateSample(attributes, state)
setq(attrs, mapValue())
setq(stateValues, mapValue('status', 'active'))
apply(func(key, value) {
setProp(attrs, key, value)
}, attributes)
apply(func(key, value) {
setProp(stateValues, key, value)
}, state)
setq(values, mapValue('ftsState', stateValues))
setq(extras, mapValue())
apply(func(key, value) {
if (or(equal(key, 'count'), equal(key, 'empty'), equal(key, 'id'), equal(key, 'label'), equal(key, 'mode'), equal(key, 'nothing'), equal(key, 'settings'), equal(key, 'tags'))) {
setProp(values, key, value)
} else {
setProp(extras, key, value)
}
}, attrs)
setProp(values, 'ftsExtraAttributes', extras)
new(Sample, values)
}, signature(array('M', 'M'), 'O')))
setq(ftsModel, mapValue('catalog', mapValue(), 'classes', array(mapValue('className', 'Sample', 'defaultState', mapValue('status', 'active'), 'key', 'instance.sample.entity', 'parentNeighborhoodKey', 'frame.sample', 'propertySchema', mapValue('additionalProperties', false, 'properties', mapValue('count', mapValue('maximum', 10, 'minimum', 0, 'type', array('integer', 'null')), 'empty', mapValue('additionalProperties', false, 'type', 'object'), 'id', mapValue('type', 'string'), 'label', mapValue('maxLength', 2, 'minLength', 1, 'type', 'string'), 'mode', mapValue('enum', array('a', 'b'), 'type', 'string'), 'nothing', mapValue('type', 'null'), 'settings', mapValue('additionalProperties', false, 'properties', mapValue('enabled', mapValue('type', 'boolean')), 'required', array('enabled'), 'type', 'object'), 'tags', mapValue('items', mapValue('type', 'string'), 'maxItems', 2, 'minItems', 1, 'type', 'array')), 'required', array('id'), 'type', 'object'), 'stateSchema', mapValue('additionalProperties', false, 'properties', mapValue('status', mapValue('enum', array('active', 'inactive'), 'type', 'string')), 'required', array('status'), 'type', 'object'))), 'descriptor', mapValue('anchorPrefix', 'fn:', 'modelName', 'validation-fixture', 'rootFrameId', 'root:test'), 'excludedInstanceEdges', 0, 'excludedInstanceFrames', 0, 'frames', array(mapValue('alias', 'root', 'canonicalId', 'root:test', 'key', 'frame.model'), mapValue('alias', 'sample', 'canonicalId', 'fn:sample', 'key', 'frame.sample')), 'hierarchy', array(), 'metadata', mapValue(), 'name', 'validation-fixture', 'relationships', array(), 'sourceNotes', array()))
