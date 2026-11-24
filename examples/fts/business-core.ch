// Generated from FTS modelseed. Use create<Class>(attributes, state) for checked construction.
class(Address) {
field('address_1', 'S')
field('address_2', 'V', null)
field('address_3', 'V', null)
field('address_id', 'S', 'address-new')
field('address_type', 'S', 'Physical')
field('city', 'V', null)
field('country_code', 'S', 'US')
field('is_primary', 'L', false)
field('owner_id', 'S')
field('owner_type', 'S')
field('postal_code', 'V', null)
field('state', 'V', null)
field('ftsState', 'M', mapValue('status', 'active'))
field('ftsExtraAttributes', 'M', mapValue())
}
setq(validateAddress, withSignature(func(attributes, state) {
setq(attrs, mapValue('address_id', 'address-new', 'address_type', 'Physical', 'country_code', 'US', 'is_primary', false))
setq(stateValues, mapValue('status', 'active'))
apply(func(key, value) {
setProp(attrs, key, value)
}, attributes)
apply(func(key, value) {
setProp(stateValues, key, value)
}, state)
if (not(or(equal(valueType(attrs), 'M')))) {
error('FTS Address.attributes: expected object')
}
if (equal(valueType(attrs), 'M')) {
setq(seen0, mapValue())
apply(func(key0, item0) {
setProp(seen0, key0, true)
if (equal(key0, 'address_1')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.address_1: expected string')
}
}
if (equal(key0, 'address_2')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.address_2: expected string')
}
}
if (equal(key0, 'address_3')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.address_3: expected string')
}
}
if (equal(key0, 'address_id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.address_id: expected string')
}
}
if (equal(key0, 'address_type')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.address_type: expected string')
}
}
if (equal(key0, 'city')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.city: expected string')
}
}
if (equal(key0, 'country_code')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.country_code: expected string')
}
}
if (equal(key0, 'is_primary')) {
if (not(or(equal(valueType(item0), 'L')))) {
error('FTS Address.attributes.is_primary: expected boolean')
}
}
if (equal(key0, 'owner_id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.owner_id: expected string')
}
}
if (equal(key0, 'owner_type')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.owner_type: expected string')
}
}
if (equal(key0, 'postal_code')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.postal_code: expected string')
}
}
if (equal(key0, 'state')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Address.attributes.state: expected string')
}
}
}, attrs)
if (not(equal(getProp(seen0, 'address_1'), true))) {
error('FTS Address.attributes.address_1: required field')
}
if (not(equal(getProp(seen0, 'address_id'), true))) {
error('FTS Address.attributes.address_id: required field')
}
if (not(equal(getProp(seen0, 'owner_id'), true))) {
error('FTS Address.attributes.owner_id: required field')
}
if (not(equal(getProp(seen0, 'owner_type'), true))) {
error('FTS Address.attributes.owner_type: required field')
}
}
if (not(or(equal(valueType(stateValues), 'M')))) {
error('FTS Address.state: expected object')
}
if (equal(valueType(stateValues), 'M')) {
setq(seen10000, mapValue())
apply(func(key10000, item10000) {
setProp(seen10000, key10000, true)
if (equal(key10000, 'status')) {
if (not(or(equal(valueType(item10000), 'S')))) {
error('FTS Address.state.status: expected string')
}
}
}, stateValues)
}
true
}, signature(array('M', 'M'), 'L')))
setq(createAddress, withSignature(func(attributes, state) {
validateAddress(attributes, state)
setq(attrs, mapValue('address_id', 'address-new', 'address_type', 'Physical', 'country_code', 'US', 'is_primary', false))
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
if (or(equal(key, 'address_1'), equal(key, 'address_2'), equal(key, 'address_3'), equal(key, 'address_id'), equal(key, 'address_type'), equal(key, 'city'), equal(key, 'country_code'), equal(key, 'is_primary'), equal(key, 'owner_id'), equal(key, 'owner_type'), equal(key, 'postal_code'), equal(key, 'state'))) {
setProp(values, key, value)
} else {
setProp(extras, key, value)
}
}, attrs)
setProp(values, 'ftsExtraAttributes', extras)
new(Address, values)
}, signature(array('M', 'M'), 'O')))
class(Organization) {
field('jurisdiction', 'V', null)
field('legalName', 'V', null)
field('org_type', 'V', null)
field('organization_id', 'S')
field('parent_id', 'V', null)
field('source_id', 'V', null)
field('source_system', 'V', null)
field('tax_id', 'V', null)
field('title', 'S', 'New Organization')
field('ftsState', 'M', mapValue('status', 'active'))
field('ftsExtraAttributes', 'M', mapValue())
}
setq(validateOrganization, withSignature(func(attributes, state) {
setq(attrs, mapValue('title', 'New Organization'))
setq(stateValues, mapValue('status', 'active'))
apply(func(key, value) {
setProp(attrs, key, value)
}, attributes)
apply(func(key, value) {
setProp(stateValues, key, value)
}, state)
if (not(or(equal(valueType(attrs), 'M')))) {
error('FTS Organization.attributes: expected object')
}
if (equal(valueType(attrs), 'M')) {
setq(seen0, mapValue())
apply(func(key0, item0) {
setProp(seen0, key0, true)
if (equal(key0, 'jurisdiction')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Organization.attributes.jurisdiction: expected string')
}
}
if (equal(key0, 'legalName')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Organization.attributes.legalName: expected string')
}
}
if (equal(key0, 'org_type')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Organization.attributes.org_type: expected string')
}
}
if (equal(key0, 'organization_id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Organization.attributes.organization_id: expected string')
}
}
if (equal(key0, 'parent_id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Organization.attributes.parent_id: expected string')
}
}
if (equal(key0, 'source_id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Organization.attributes.source_id: expected string')
}
}
if (equal(key0, 'source_system')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Organization.attributes.source_system: expected string')
}
}
if (equal(key0, 'tax_id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Organization.attributes.tax_id: expected string')
}
}
if (equal(key0, 'title')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Organization.attributes.title: expected string')
}
}
}, attrs)
if (not(equal(getProp(seen0, 'title'), true))) {
error('FTS Organization.attributes.title: required field')
}
if (not(equal(getProp(seen0, 'organization_id'), true))) {
error('FTS Organization.attributes.organization_id: required field')
}
}
if (not(or(equal(valueType(stateValues), 'M')))) {
error('FTS Organization.state: expected object')
}
if (equal(valueType(stateValues), 'M')) {
setq(seen10000, mapValue())
apply(func(key10000, item10000) {
setProp(seen10000, key10000, true)
if (equal(key10000, 'status')) {
if (not(or(equal(valueType(item10000), 'S')))) {
error('FTS Organization.state.status: expected string')
}
}
}, stateValues)
}
true
}, signature(array('M', 'M'), 'L')))
setq(createOrganization, withSignature(func(attributes, state) {
validateOrganization(attributes, state)
setq(attrs, mapValue('title', 'New Organization'))
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
if (or(equal(key, 'jurisdiction'), equal(key, 'legalName'), equal(key, 'org_type'), equal(key, 'organization_id'), equal(key, 'parent_id'), equal(key, 'source_id'), equal(key, 'source_system'), equal(key, 'tax_id'), equal(key, 'title'))) {
setProp(values, key, value)
} else {
setProp(extras, key, value)
}
}, attrs)
setProp(values, 'ftsExtraAttributes', extras)
new(Organization, values)
}, signature(array('M', 'M'), 'O')))
class(Person) {
field('created_at', 'V', null)
field('deceased_at', 'V', null)
field('deleted_at', 'V', null)
field('dob', 'S')
field('first_name', 'S', 'New')
field('gender', 'V', null)
field('language', 'V', null)
field('last_name', 'S', 'Person')
field('marital_status', 'V', null)
field('middle_name', 'V', null)
field('person_id', 'S')
field('prefix', 'V', null)
field('suffix', 'V', null)
field('time_zone', 'V', null)
field('updated_at', 'V', null)
field('ftsState', 'M', mapValue('status', 'active'))
field('ftsExtraAttributes', 'M', mapValue())
}
setq(validatePerson, withSignature(func(attributes, state) {
setq(attrs, mapValue('first_name', 'New', 'last_name', 'Person'))
setq(stateValues, mapValue('status', 'active'))
apply(func(key, value) {
setProp(attrs, key, value)
}, attributes)
apply(func(key, value) {
setProp(stateValues, key, value)
}, state)
if (not(or(equal(valueType(attrs), 'M')))) {
error('FTS Person.attributes: expected object')
}
if (equal(valueType(attrs), 'M')) {
setq(seen0, mapValue())
apply(func(key0, item0) {
setProp(seen0, key0, true)
if (equal(key0, 'created_at')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.created_at: expected string')
}
}
if (equal(key0, 'deceased_at')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.deceased_at: expected string')
}
}
if (equal(key0, 'deleted_at')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.deleted_at: expected string')
}
}
if (equal(key0, 'dob')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.dob: expected string')
}
}
if (equal(key0, 'first_name')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.first_name: expected string')
}
}
if (equal(key0, 'gender')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.gender: expected string')
}
}
if (equal(key0, 'language')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.language: expected string')
}
}
if (equal(key0, 'last_name')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.last_name: expected string')
}
}
if (equal(key0, 'marital_status')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.marital_status: expected string')
}
}
if (equal(key0, 'middle_name')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.middle_name: expected string')
}
}
if (equal(key0, 'person_id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.person_id: expected string')
}
}
if (equal(key0, 'prefix')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.prefix: expected string')
}
}
if (equal(key0, 'suffix')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.suffix: expected string')
}
}
if (equal(key0, 'time_zone')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.time_zone: expected string')
}
}
if (equal(key0, 'updated_at')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Person.attributes.updated_at: expected string')
}
}
}, attrs)
if (not(equal(getProp(seen0, 'dob'), true))) {
error('FTS Person.attributes.dob: required field')
}
if (not(equal(getProp(seen0, 'first_name'), true))) {
error('FTS Person.attributes.first_name: required field')
}
if (not(equal(getProp(seen0, 'last_name'), true))) {
error('FTS Person.attributes.last_name: required field')
}
if (not(equal(getProp(seen0, 'person_id'), true))) {
error('FTS Person.attributes.person_id: required field')
}
}
if (not(or(equal(valueType(stateValues), 'M')))) {
error('FTS Person.state: expected object')
}
if (equal(valueType(stateValues), 'M')) {
setq(seen10000, mapValue())
apply(func(key10000, item10000) {
setProp(seen10000, key10000, true)
if (equal(key10000, 'status')) {
if (not(or(equal(valueType(item10000), 'S')))) {
error('FTS Person.state.status: expected string')
}
}
}, stateValues)
}
true
}, signature(array('M', 'M'), 'L')))
setq(createPerson, withSignature(func(attributes, state) {
validatePerson(attributes, state)
setq(attrs, mapValue('first_name', 'New', 'last_name', 'Person'))
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
if (or(equal(key, 'created_at'), equal(key, 'deceased_at'), equal(key, 'deleted_at'), equal(key, 'dob'), equal(key, 'first_name'), equal(key, 'gender'), equal(key, 'language'), equal(key, 'last_name'), equal(key, 'marital_status'), equal(key, 'middle_name'), equal(key, 'person_id'), equal(key, 'prefix'), equal(key, 'suffix'), equal(key, 'time_zone'), equal(key, 'updated_at'))) {
setProp(values, key, value)
} else {
setProp(extras, key, value)
}
}, attrs)
setProp(values, 'ftsExtraAttributes', extras)
new(Person, values)
}, signature(array('M', 'M'), 'O')))
class(Phone) {
field('area_code', 'V', null)
field('can_fax', 'L', false)
field('can_sms', 'L', false)
field('can_voice', 'L', true)
field('country_code', 'V', null)
field('extension', 'V', null)
field('is_primary', 'V', null)
field('number', 'V', null)
field('owner_id', 'S')
field('owner_type', 'S')
field('phone_id', 'S')
field('phone_type', 'V', null)
field('ftsState', 'M', mapValue('status', 'active'))
field('ftsExtraAttributes', 'M', mapValue())
}
setq(validatePhone, withSignature(func(attributes, state) {
setq(attrs, mapValue('can_fax', false, 'can_sms', false, 'can_voice', true))
setq(stateValues, mapValue('status', 'active'))
apply(func(key, value) {
setProp(attrs, key, value)
}, attributes)
apply(func(key, value) {
setProp(stateValues, key, value)
}, state)
if (not(or(equal(valueType(attrs), 'M')))) {
error('FTS Phone.attributes: expected object')
}
if (equal(valueType(attrs), 'M')) {
setq(seen0, mapValue())
apply(func(key0, item0) {
setProp(seen0, key0, true)
if (equal(key0, 'area_code')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Phone.attributes.area_code: expected string')
}
}
if (equal(key0, 'can_fax')) {
if (not(or(equal(valueType(item0), 'L')))) {
error('FTS Phone.attributes.can_fax: expected boolean')
}
}
if (equal(key0, 'can_sms')) {
if (not(or(equal(valueType(item0), 'L')))) {
error('FTS Phone.attributes.can_sms: expected boolean')
}
}
if (equal(key0, 'can_voice')) {
if (not(or(equal(valueType(item0), 'L')))) {
error('FTS Phone.attributes.can_voice: expected boolean')
}
}
if (equal(key0, 'country_code')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Phone.attributes.country_code: expected string')
}
}
if (equal(key0, 'extension')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Phone.attributes.extension: expected string')
}
}
if (equal(key0, 'is_primary')) {
if (not(or(equal(valueType(item0), 'L')))) {
error('FTS Phone.attributes.is_primary: expected boolean')
}
}
if (equal(key0, 'number')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Phone.attributes.number: expected string')
}
}
if (equal(key0, 'owner_id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Phone.attributes.owner_id: expected string')
}
}
if (equal(key0, 'owner_type')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Phone.attributes.owner_type: expected string')
}
}
if (equal(key0, 'phone_id')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Phone.attributes.phone_id: expected string')
}
}
if (equal(key0, 'phone_type')) {
if (not(or(equal(valueType(item0), 'S')))) {
error('FTS Phone.attributes.phone_type: expected string')
}
}
}, attrs)
if (not(equal(getProp(seen0, 'owner_id'), true))) {
error('FTS Phone.attributes.owner_id: required field')
}
if (not(equal(getProp(seen0, 'owner_type'), true))) {
error('FTS Phone.attributes.owner_type: required field')
}
if (not(equal(getProp(seen0, 'phone_id'), true))) {
error('FTS Phone.attributes.phone_id: required field')
}
}
if (not(or(equal(valueType(stateValues), 'M')))) {
error('FTS Phone.state: expected object')
}
if (equal(valueType(stateValues), 'M')) {
setq(seen10000, mapValue())
apply(func(key10000, item10000) {
setProp(seen10000, key10000, true)
if (equal(key10000, 'status')) {
if (not(or(equal(valueType(item10000), 'S')))) {
error('FTS Phone.state.status: expected string')
}
}
}, stateValues)
}
true
}, signature(array('M', 'M'), 'L')))
setq(createPhone, withSignature(func(attributes, state) {
validatePhone(attributes, state)
setq(attrs, mapValue('can_fax', false, 'can_sms', false, 'can_voice', true))
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
if (or(equal(key, 'area_code'), equal(key, 'can_fax'), equal(key, 'can_sms'), equal(key, 'can_voice'), equal(key, 'country_code'), equal(key, 'extension'), equal(key, 'is_primary'), equal(key, 'number'), equal(key, 'owner_id'), equal(key, 'owner_type'), equal(key, 'phone_id'), equal(key, 'phone_type'))) {
setProp(values, key, value)
} else {
setProp(extras, key, value)
}
}, attrs)
setProp(values, 'ftsExtraAttributes', extras)
new(Phone, values)
}, signature(array('M', 'M'), 'O')))
setq(ftsModel, mapValue('catalog', mapValue('compiledModelId', 'compiled.business-core.base', 'description', 'Shared business identity and contact model for organizations, people, addresses, phones, and their relationships.', 'domain', 'business-core', 'executionMode', 'modeling', 'modelSetId', 'mset.business-core.base', 'modelSetName', 'Business Core Base', 'modelSpecId', 'mspec.business-core.base', 'modelSpecName', 'Business Core Base', 'rootFrameId', 'froot:business-core', 'sourceDocument', 'docs/CoreBusinessFramesTheory.md', 'sourceKind', 'seed', 'storageMode', 'memory'), 'classes', array(mapValue('className', 'Address', 'defaultAttributes', mapValue('address_id', 'address-new', 'address_type', 'Physical', 'country_code', 'US', 'is_primary', false), 'defaultState', mapValue('status', 'active'), 'description', 'Polymorphically owned address record archetype.', 'id', 'iar.business-core.address.record', 'key', 'instance.address.record', 'name', 'Address', 'parentNeighborhoodKey', 'frame.address', 'propertySchema', mapValue('properties', mapValue('address_1', mapValue('title', 'Address line 1', 'type', 'string'), 'address_2', mapValue('title', 'Address line 2', 'type', 'string'), 'address_3', mapValue('title', 'Address line 3', 'type', 'string'), 'address_id', mapValue('title', 'Address ID', 'type', 'string'), 'address_type', mapValue('title', 'Address type', 'type', 'string'), 'city', mapValue('title', 'City', 'type', 'string'), 'country_code', mapValue('title', 'Country code', 'type', 'string'), 'is_primary', mapValue('title', 'Is primary', 'type', 'boolean'), 'owner_id', mapValue('title', 'Owner ID', 'type', 'string'), 'owner_type', mapValue('title', 'Owner type', 'type', 'string'), 'postal_code', mapValue('title', 'Postal code', 'type', 'string'), 'state', mapValue('title', 'State', 'type', 'string')), 'required', array('address_1', 'address_id', 'owner_id', 'owner_type'), 'type', 'object'), 'stateSchema', mapValue('properties', mapValue('status', mapValue('title', 'Status', 'type', 'string')), 'required', null, 'type', 'object'), 'x-relational', mapValue('mysql', mapValue('columns', mapValue('address_1', mapValue('column', 'Address1', 'index', true, 'nullable', false, 'sqlType', 'varchar(50)'), 'address_2', mapValue('column', 'Address2', 'nullable', true, 'sqlType', 'varchar(50)'), 'address_3', mapValue('column', 'Address3', 'nullable', true, 'sqlType', 'varchar(50)'), 'address_id', mapValue('column', 'AddressID', 'nullable', false, 'primaryKey', true, 'sqlType', 'char(10)'), 'address_type', mapValue('column', 'AddressType', 'index', true, 'nullable', true, 'sqlType', 'varchar(10)'), 'city', mapValue('column', 'City', 'nullable', false, 'sqlType', 'varchar(50)'), 'country_code', mapValue('column', 'CountyCode', 'index', true, 'nullable', false, 'sqlType', 'char(2)'), 'is_primary', mapValue('column', 'IsPrimary', 'index', true, 'nullable', false, 'sqlType', 'boolean'), 'owner_id', mapValue('column', 'OwnerID', 'index', true, 'nullable', false, 'sqlType', 'char(10)'), 'owner_type', mapValue('column', 'OwnerType', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'postal_code', mapValue('column', 'PostalCode', 'nullable', false, 'sqlType', 'varchar(10)'), 'state', mapValue('column', 'State', 'nullable', false, 'sqlType', 'char(2)')), 'primaryKey', mapValue('fields', array('AddressID'), 'strategy', 'generated'), 'table', 'Address'))), mapValue('className', 'Organization', 'defaultAttributes', mapValue('title', 'New Organization'), 'defaultState', mapValue('status', 'active'), 'description', 'Organization identity archetype.', 'id', 'iar.business-core.organization.entity', 'key', 'instance.organization.entity', 'name', 'Organization', 'parentNeighborhoodKey', 'frame.organization', 'propertySchema', mapValue('properties', mapValue('jurisdiction', mapValue('title', 'Jurisdiction', 'type', 'string'), 'legalName', mapValue('title', 'Legal name', 'type', 'string'), 'org_type', mapValue('title', 'Organization type', 'type', 'string'), 'organization_id', mapValue('title', 'Organization ID', 'type', 'string'), 'parent_id', mapValue('title', 'Parent ID', 'type', 'string'), 'source_id', mapValue('title', 'Source ID', 'type', 'string'), 'source_system', mapValue('title', 'Source system', 'type', 'string'), 'tax_id', mapValue('title', 'Tax identifier', 'type', 'string'), 'title', mapValue('title', 'Title', 'type', 'string')), 'required', array('title', 'organization_id'), 'type', 'object'), 'stateSchema', mapValue('properties', mapValue('status', mapValue('title', 'Status', 'type', 'string')), 'required', null, 'type', 'object'), 'x-relational', mapValue('mysql', mapValue('columns', mapValue('jurisdiction', mapValue('column', 'Jurisdiction', 'nullable', true, 'sqlType', 'varchar(50)'), 'legalName', mapValue('column', 'LegalName', 'nullable', true, 'sqlType', 'varchar(50)'), 'org_type', mapValue('column', 'OrgType', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'organization_id', mapValue('column', 'OrgID', 'nullable', false, 'primaryKey', true, 'sqlType', 'char(10)'), 'parent_id', mapValue('column', 'ParentID', 'index', true, 'nullable', true, 'sqlType', 'char(10)'), 'source_id', mapValue('column', 'SourceID', 'index', true, 'nullable', true, 'sqlType', 'varchar(20)'), 'source_system', mapValue('column', 'SourceSystem', 'index', true, 'nullable', true, 'sqlType', 'varchar(20)'), 'tax_id', mapValue('column', 'TaxID', 'index', true, 'nullable', true, 'sqlType', 'varchar(20)'), 'title', mapValue('column', 'Title', 'index', true, 'nullable', false, 'sqlType', 'varchar(50)', 'unique', true)), 'primaryKey', mapValue('fields', array('organization_id'), 'strategy', 'generated'), 'table', 'Organization'))), mapValue('className', 'Person', 'defaultAttributes', mapValue('first_name', 'New', 'last_name', 'Person'), 'defaultState', mapValue('status', 'active'), 'description', 'Natural person identity archetype.', 'id', 'iar.business-core.person.entity', 'key', 'instance.person.entity', 'name', 'Person', 'parentNeighborhoodKey', 'frame.person', 'propertySchema', mapValue('properties', mapValue('created_at', mapValue('title', 'Created at', 'type', 'string'), 'deceased_at', mapValue('title', 'Deceased at', 'type', 'string'), 'deleted_at', mapValue('title', 'Deleted at', 'type', 'string'), 'dob', mapValue('description', 'Birth date', 'title', 'Date of birth', 'type', 'string'), 'first_name', mapValue('description', 'Given name', 'title', 'First name', 'type', 'string'), 'gender', mapValue('description', 'Gender', 'title', 'Gender', 'type', 'string'), 'language', mapValue('description', 'Preferred language', 'title', 'Language', 'type', 'string'), 'last_name', mapValue('description', 'Surname', 'title', 'Last name', 'type', 'string'), 'marital_status', mapValue('description', 'Married, Single, Divorced', 'title', 'Marital status', 'type', 'string'), 'middle_name', mapValue('description', 'Middle name', 'title', 'Middle name', 'type', 'string'), 'person_id', mapValue('description', 'Primary key', 'title', 'Person ID', 'type', 'string'), 'prefix', mapValue('description', 'Mr, Ms, Mrs, Dr', 'title', 'Prefix', 'type', 'string'), 'suffix', mapValue('description', 'Lineage or profession suffix', 'title', 'Suffix', 'type', 'string'), 'time_zone', mapValue('description', 'Time zone', 'title', 'Time zone', 'type', 'string'), 'updated_at', mapValue('title', 'Updated at', 'type', 'string')), 'required', array('dob', 'first_name', 'last_name', 'person_id'), 'type', 'object'), 'stateSchema', mapValue('properties', mapValue('status', mapValue('title', 'Status', 'type', 'string')), 'required', null, 'type', 'object'), 'x-relational', mapValue('mysql', mapValue('columns', mapValue('created_at', mapValue('column', 'CreatedAt', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'deceased_at', mapValue('column', 'DeceasedAt', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'deleted_at', mapValue('column', 'DeletedAt', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'dob', mapValue('column', 'DOB', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'first_name', mapValue('column', 'FirstName', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'gender', mapValue('column', 'Gender', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'language', mapValue('column', 'Language', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'last_name', mapValue('column', 'LastName', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'marital_status', mapValue('column', 'MaritalStatus', 'index', true, 'nullable', true, 'sqlType', 'varchar(10)'), 'middle_name', mapValue('column', 'MiddleName', 'nullable', true, 'sqlType', 'varchar(20)'), 'person_id', mapValue('column', 'PersonID', 'nullable', false, 'primaryKey', true, 'sqlType', 'char(10)'), 'prefix', mapValue('column', 'Prefix', 'nullable', true, 'sqlType', 'varchar(10)'), 'suffix', mapValue('column', 'Suffix', 'nullable', true, 'sqlType', 'varchar(10)'), 'time_zone', mapValue('column', 'TimeZone', 'index', true, 'nullable', false, 'sqlType', 'varchar(10)'), 'updated_at', mapValue('column', 'UpdatedAt', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)')), 'primaryKey', mapValue('fields', array('PersonID'), 'strategy', 'generated'), 'table', 'Person'))), mapValue('className', 'Phone', 'defaultAttributes', mapValue('can_fax', false, 'can_sms', false, 'can_voice', true), 'defaultState', mapValue('status', 'active'), 'description', 'Polymorphically owned telephone endpoint archetype.', 'id', 'iar.business-core.phone.record', 'key', 'instance.phone.record', 'name', 'Phone', 'parentNeighborhoodKey', 'frame.phone', 'propertySchema', mapValue('properties', mapValue('area_code', mapValue('title', 'Area Code', 'type', 'string'), 'can_fax', mapValue('title', 'Can receive fax', 'type', 'boolean'), 'can_sms', mapValue('title', 'Can receive SMS', 'type', 'boolean'), 'can_voice', mapValue('title', 'Can receive voice', 'type', 'boolean'), 'country_code', mapValue('title', 'Country calling code', 'type', 'string'), 'extension', mapValue('title', 'Extension', 'type', 'string'), 'is_primary', mapValue('title', 'Is primary', 'type', 'boolean'), 'number', mapValue('title', 'Number', 'type', 'string'), 'owner_id', mapValue('title', 'Owner ID', 'type', 'string'), 'owner_type', mapValue('title', 'Owner type', 'type', 'string'), 'phone_id', mapValue('title', 'Phone ID', 'type', 'string'), 'phone_type', mapValue('title', 'Phone type', 'type', 'string')), 'required', array('owner_id', 'owner_type', 'phone_id'), 'type', 'object'), 'stateSchema', mapValue('properties', mapValue('status', mapValue('title', 'Status', 'type', 'string')), 'required', null, 'type', 'object'), 'x-relational', mapValue('mysql', mapValue('columns', mapValue('area_code', mapValue('column', 'AreaCode', 'index', true, 'nullable', false, 'sqlType', 'varchar(3)'), 'can_fax', mapValue('column', 'CanFAX', 'index', true, 'nullable', false, 'sqlType', 'boolean'), 'can_sms', mapValue('column', 'CanSMS', 'index', true, 'nullable', false, 'sqlType', 'boolean'), 'can_voice', mapValue('column', 'CanVoice', 'index', true, 'nullable', false, 'sqlType', 'boolean'), 'country_code', mapValue('column', 'CountryCode', 'index', true, 'nullable', false, 'sqlType', 'varchar(3)'), 'extension', mapValue('column', 'Extension', 'nullable', true, 'sqlType', 'varchar(10)'), 'is_primary', mapValue('column', 'IsPrimary', 'nullable', false, 'sqlType', 'boolean'), 'number', mapValue('column', 'Number', 'index', true, 'nullable', false, 'sqlType', 'varchar(10)'), 'owner_id', mapValue('column', 'OwnerID', 'index', true, 'nullable', false, 'sqlType', 'char(10)'), 'owner_type', mapValue('column', 'OwnerType', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'phone_id', mapValue('column', 'PhoneID', 'nullable', false, 'primaryKey', true, 'sqlType', 'char(10)'), 'phone_type', mapValue('column', 'PhoneType', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)')), 'primaryKey', mapValue('fields', array('phone_id'), 'strategy', 'generated'), 'table', 'Phone')))), 'descriptor', mapValue('anchorPrefix', 'fn:', 'capabilities', mapValue('asyncQueryJobs', false, 'modeling', true, 'productionQueries', false, 'productionWrites', false), 'frameCatalogGroupLabel', 'Core Business Frames', 'frameCatalogLabelMode', 'name', 'legend', mapValue('anchor', 'Shared business frame', 'instance', 'Seeded example entity', 'root', 'Business core root'), 'modelName', 'business-core', 'rootFrameId', 'froot:business-core'), 'excludedInstanceEdges', 13, 'excludedInstanceFrames', 7, 'frames', array(mapValue('alias', 'business_core_root', 'canonicalId', 'froot:business-core', 'description', 'Root frame for reusable organization, person, address, and phone semantics.', 'key', 'frame.model.business_core', 'name', 'Business Core', 'typeName', 'frame.model'), mapValue('alias', 'organization', 'canonicalId', 'fn:organization', 'description', 'Legally, economically, administratively, or operationally recognized collective entities.', 'key', 'frame.organization', 'name', 'Organization Frame', 'typeName', 'neighborhood'), mapValue('alias', 'person', 'canonicalId', 'fn:person', 'description', 'Natural persons independent of organization-specific roles.', 'key', 'frame.person', 'name', 'Person Frame', 'typeName', 'neighborhood'), mapValue('alias', 'address', 'canonicalId', 'fn:address', 'description', 'Reusable address records owned polymorphically by organizations, people, or other frames.', 'key', 'frame.address', 'name', 'Address Frame', 'typeName', 'neighborhood'), mapValue('alias', 'phone', 'canonicalId', 'fn:phone', 'description', 'Reusable telephone endpoint records owned polymorphically by organizations, people, or other frames.', 'key', 'frame.phone', 'name', 'Phone Frame', 'typeName', 'neighborhood')), 'hierarchy', array(mapValue('description', 'Connects the business core root to the address frame.', 'fromAlias', 'business_core_root', 'key', 'edge.business_core_contains_address', 'name', 'Business core contains address', 'toAlias', 'address', 'typeName', 'contains'), mapValue('description', 'Connects the business core root to the organization frame.', 'fromAlias', 'business_core_root', 'key', 'edge.business_core_contains_organization', 'name', 'Business core contains organization', 'toAlias', 'organization', 'typeName', 'contains'), mapValue('description', 'Connects the business core root to the person frame.', 'fromAlias', 'business_core_root', 'key', 'edge.business_core_contains_person', 'name', 'Business core contains person', 'toAlias', 'person', 'typeName', 'contains'), mapValue('description', 'Connects the business core root to the phone frame.', 'fromAlias', 'business_core_root', 'key', 'edge.business_core_contains_phone', 'name', 'Business core contains phone', 'toAlias', 'phone', 'typeName', 'contains')), 'metadata', mapValue(), 'name', 'business-core-neighborhood', 'relationships', array(mapValue('description', 'Connects an organization to an address it owns.', 'fromAlias', 'organization', 'key', 'edge.organization_has_address', 'name', 'Organization has address', 'sourceArchetypeKey', 'instance.organization.entity', 'sourceClass', 'Organization', 'targetArchetypeKey', 'instance.address.record', 'targetClass', 'Address', 'toAlias', 'address', 'typeName', 'has_address'), mapValue('description', 'Connects an organization to a person through an association table carrying role and relationship attributes.', 'fromAlias', 'organization', 'key', 'edge.organization_has_person', 'name', 'Organization has person', 'sourceArchetypeKey', 'instance.organization.entity', 'sourceClass', 'Organization', 'targetArchetypeKey', 'instance.person.entity', 'targetClass', 'Person', 'toAlias', 'person', 'typeName', 'has_person', 'x-relational', mapValue('mysql', mapValue('associationTable', mapValue('indexes', array(mapValue('fields', array('PersonID'), 'name', 'idx_organization_person_person'), mapValue('fields', array('RelationType'), 'name', 'idx_organization_person_relationship_type'), mapValue('fields', array('EmployeeID'), 'name', 'idx_organization_person_employee_identifier')), 'payloadColumns', mapValue('department', mapValue('column', 'Department', 'nullable', true, 'sqlType', 'varchar(50)'), 'effective_from', mapValue('column', 'EffectiveFrom', 'nullable', true, 'sqlType', 'date'), 'effective_through', mapValue('column', 'EffectiveThrough', 'nullable', true, 'sqlType', 'date'), 'employee_id', mapValue('column', 'EmployeeID', 'index', true, 'nullable', true, 'sqlType', 'varchar(30)'), 'is_primary', mapValue('column', 'IsPrimary', 'nullable', false, 'sqlType', 'boolean'), 'org_person_id', mapValue('column', 'OrgPersonID', 'nullable', false, 'primaryKey', true, 'sqlType', 'char(10)'), 'relation_type', mapValue('column', 'RelationType', 'index', true, 'nullable', false, 'sqlType', 'varchar(20)'), 'role', mapValue('column', 'Role', 'nullable', true, 'sqlType', 'varchar(20)'), 'title', mapValue('column', 'Title', 'nullable', true, 'sqlType', 'varchar(30)')), 'primaryKeyFields', array('OrgPersonID'), 'sourceArchetypeKey', 'instance.organization.entity', 'sourceColumn', 'OrgID', 'sourceReferencedColumn', 'OrgID', 'table', 'OrganizationPerson', 'targetArchetypeKey', 'instance.person.entity', 'targetColumn', 'PersonID', 'targetReferencedColumn', 'PersonID'), 'mode', 'association-table'))), mapValue('description', 'Connects an organization to a phone number it owns.', 'fromAlias', 'organization', 'key', 'edge.organization_has_phone', 'name', 'Organization has phone', 'sourceArchetypeKey', 'instance.organization.entity', 'sourceClass', 'Organization', 'targetArchetypeKey', 'instance.phone.record', 'targetClass', 'Phone', 'toAlias', 'phone', 'typeName', 'has_phone'), mapValue('description', 'Connects a parent organization to its child organization.', 'fromAlias', 'organization', 'key', 'edge.organization_parent_of_organization', 'name', 'Organization parent of organization', 'sourceArchetypeKey', 'instance.organization.entity', 'sourceClass', 'Organization', 'targetArchetypeKey', 'instance.organization.entity', 'targetClass', 'Organization', 'toAlias', 'organization', 'typeName', 'parent_of'), mapValue('description', 'Connects a person to an address they own.', 'fromAlias', 'person', 'key', 'edge.person_has_address', 'name', 'Person has address', 'sourceArchetypeKey', 'instance.person.entity', 'sourceClass', 'Person', 'targetArchetypeKey', 'instance.address.record', 'targetClass', 'Address', 'toAlias', 'address', 'typeName', 'has_address'), mapValue('description', 'Connects a person to a phone number they own.', 'fromAlias', 'person', 'key', 'edge.person_has_phone', 'name', 'Person has phone', 'sourceArchetypeKey', 'instance.person.entity', 'sourceClass', 'Person', 'targetArchetypeKey', 'instance.phone.record', 'targetClass', 'Phone', 'toAlias', 'phone', 'typeName', 'has_phone')), 'sourceNotes', array('Generated from docs/CoreBusinessFramesTheory.md as the shared business identity and contact foundation for composed domain models.')))
