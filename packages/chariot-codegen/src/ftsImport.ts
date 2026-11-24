import { generateChariotSourceFromAST, type ChariotASTNode as AST } from './chariotAST';

type JSONValue = null | boolean | number | string | JSONValue[] | JSONObject;
interface JSONObject { [key: string]: JSONValue }
const hasOwn = (value: object, key: string): boolean => Object.prototype.hasOwnProperty.call(value, key);
interface Schema {
  raw: JSONObject;
  types: string[];
  properties: Record<string, Schema>;
  required: string[];
  items?: Schema;
}
interface Archetype {
  raw: JSONObject;
  key: string;
  name: string;
  parent: string;
  properties: Schema;
  state: Schema;
  defaults: JSONObject;
  stateDefaults: JSONObject;
}
export interface FTSImportResult {
  source: string;
  ast: AST;
  classNames: string[];
  relationshipKeys: string[];
  excludedInstanceFrames: number;
  excludedInstanceEdges: number;
}

function fail(path: string, message: string): never {
  throw new Error(`FTS ${path}: ${message}`);
}
function object(value: unknown, path: string): JSONObject {
  if (!value || typeof value !== 'object' || Array.isArray(value)) fail(path, 'expected an object');
  return value as JSONObject;
}
function json(value: unknown, path = '$', active = new Set<object>()): JSONValue {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') return value;
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value !== 'object' || !value) fail(path, 'expected finite JSON data');
  if (active.has(value)) fail(path, 'cyclic input');
  active.add(value);
  const result: JSONValue = Array.isArray(value)
    ? value.map((v, i) => json(v, `${path}[${i}]`, active))
    : Object.fromEntries(Object.entries(value).map(([k, v]) => [k, json(v, `${path}.${k}`, active)]));
  active.delete(value);
  return result;
}
function text(value: unknown, path: string): string {
  if (typeof value !== 'string' || !value.trim()) fail(path, 'expected a nonempty string');
  return value;
}
function array(value: unknown, path: string): JSONValue[] {
  if (!Array.isArray(value)) fail(path, 'expected an array');
  return value;
}
function entries(model: JSONObject, field: string): JSONObject[] {
  return array(model[field] ?? [], field).map((v, i) => object(v, `${field}[${i}]`));
}
function stable(value: JSONValue): string {
  if (Array.isArray(value)) return `[${value.map(stable).join(',')}]`;
  if (value && typeof value === 'object') {
    return `{${Object.keys(value).sort().map(k => `${JSON.stringify(k)}:${stable(value[k])}`).join(',')}}`;
  }
  return JSON.stringify(value);
}
function unique(items: JSONObject[], field: string, kind: string): Map<string, JSONObject> {
  const result = new Map<string, JSONObject>();
  for (const item of items) {
    const key = text(item[field], `${kind}.${field}`);
    if (result.has(key)) fail(kind, `duplicate ${field} "${key}"`);
    result.set(key, item);
  }
  return result;
}

/** Filesystem-independent composition matching FTS's import order and conflict rules. */
export function loadFTSModel(
  root: string,
  read: (id: string) => unknown,
  resolve: (parent: string, source: string) => string,
): JSONObject {
  const active = new Set<string>();
  function visit(id: string): JSONObject {
    if (active.has(id)) fail(id, 'modelseed import cycle');
    active.add(id);
    try {
      const local = object(json(read(id)), id);
      const parts = entries(local, 'imports').map(item =>
        visit(resolve(id, text(item.source, `${id}.imports.source`))));
      parts.push(local);
      const merged: JSONObject = { frames: [], edges: [], archetypes: [], sourceNotes: [], metadata: {} };
      for (const part of parts) {
        for (const field of ['name', 'descriptor', 'catalog', 'x-relational']) {
          if (part[field] !== undefined && part[field] !== '' &&
              !(typeof part[field] === 'object' && part[field] !== null && Object.keys(part[field]).length === 0)) {
            merged[field] = part[field];
          }
        }
        for (const [field, keyField] of [['frames', 'alias'], ['edges', 'key'], ['archetypes', 'key']]) {
          const current = entries(merged, field);
          const index = unique(current, keyField, field);
          for (const item of entries(part, field)) {
            const key = text(item[keyField], `${id}.${field}.${keyField}`);
            const previous = index.get(key);
            if (previous && stable(previous) !== stable(item)) fail(id, `${field} key "${key}" conflicts`);
            if (!previous) { current.push(item); index.set(key, item); }
          }
          merged[field] = current;
        }
        merged.metadata = {
          ...object(merged.metadata, 'metadata'),
          ...object(part.metadata ?? {}, `${id}.metadata`),
        };
        merged.sourceNotes = [...array(merged.sourceNotes, 'sourceNotes'), ...array(part.sourceNotes ?? [], `${id}.sourceNotes`)];
      }
      return merged;
    } finally {
      active.delete(id);
    }
  }
  return visit(root);
}

const schemaAnnotations = ['title', 'description', '$comment', '$schema', '$id', 'default', 'examples'];
const schemaKeywords = [...schemaAnnotations, 'type', 'properties', 'required', 'additionalProperties', 'items',
  'enum', 'minimum', 'maximum', 'minLength', 'maxLength', 'minItems', 'maxItems'];
const typeCodes: Record<string, string> = {
  string: 'S', boolean: 'L', number: 'N', integer: 'N', object: 'M', array: 'A',
};

function schema(value: unknown, path: string): Schema {
  const raw = object(value, path);
  for (const key of Object.keys(raw)) {
    if (!schemaKeywords.includes(key)) fail(`${path}.${key}`, 'unsupported schema keyword');
  }
  const types = (Array.isArray(raw.type) ? raw.type : [raw.type]).map((v, i) => text(v, `${path}.type[${i}]`));
  if (!types.length || new Set(types).size !== types.length || types.some(t => !hasOwn(typeCodes, t) && t !== 'null')) {
    fail(path, 'unsupported or duplicate schema type');
  }
  const properties = Object.fromEntries(Object.entries(object(raw.properties === undefined ? {} : raw.properties, `${path}.properties`))
    .map(([k, v]) => [k, schema(v, `${path}.properties.${k}`)]));
  const required = array(raw.required ?? [], `${path}.required`).map(v => text(v, `${path}.required`));
  if (new Set(required).size !== required.length || required.some(k => !hasOwn(properties, k))) {
    fail(path, 'required fields must be unique declared properties');
  }
  if (raw.additionalProperties !== undefined && typeof raw.additionalProperties !== 'boolean') {
    fail(path, 'only boolean additionalProperties is supported');
  }
  for (const key of ['minimum', 'maximum', 'minLength', 'maxLength', 'minItems', 'maxItems']) {
    const n = raw[key];
    if (n !== undefined && (typeof n !== 'number' || (key !== 'minimum' && key !== 'maximum' && (!Number.isInteger(n) || n < 0)))) {
      fail(`${path}.${key}`, 'invalid numeric constraint');
    }
  }
  for (const [min, max] of [['minimum', 'maximum'], ['minLength', 'maxLength'], ['minItems', 'maxItems']]) {
    if (typeof raw[min] === 'number' && typeof raw[max] === 'number' && raw[min] > raw[max]) fail(path, `${min} exceeds ${max}`);
  }
  if (raw.enum !== undefined) {
    const values = array(raw.enum, `${path}.enum`);
    if (!values.length || values.some(v => v !== null && typeof v === 'object')) fail(path, 'enum supports nonempty primitive values only');
  }
  if (!types.includes('object') && ['properties', 'required', 'additionalProperties'].some(k => raw[k] !== undefined)) {
    fail(path, 'object constraints require object type');
  }
  if (!types.includes('array') && ['items', 'minItems', 'maxItems'].some(k => raw[k] !== undefined)) fail(path, 'array constraints require array type');
  if (!types.includes('string') && ['minLength', 'maxLength'].some(k => raw[k] !== undefined)) fail(path, 'string constraints require string type');
  if (!types.some(t => t === 'number' || t === 'integer') && ['minimum', 'maximum'].some(k => raw[k] !== undefined)) {
    fail(path, 'numeric constraints require numeric type');
  }
  return { raw, types, properties, required, items: raw.items === undefined ? undefined : schema(raw.items, `${path}.items`) };
}

function validateJSON(value: JSONValue, s: Schema, path: string): void {
  const matches = s.types.some(t => t === 'null' ? value === null :
    t === 'array' ? Array.isArray(value) : t === 'object' ? value !== null && typeof value === 'object' && !Array.isArray(value) :
    t === 'integer' ? typeof value === 'number' && Number.isInteger(value) : typeof value === t);
  if (!matches) fail(path, `expected ${s.types.join('|')}`);
  if (s.raw.enum && !array(s.raw.enum, path).some(v => stable(v) === stable(value))) fail(path, 'not in enum');
  if (value === null) return;
  if (typeof value === 'number') {
    if (typeof s.raw.minimum === 'number' && value < s.raw.minimum) fail(path, 'below minimum');
    if (typeof s.raw.maximum === 'number' && value > s.raw.maximum) fail(path, 'above maximum');
  }
  const length = typeof value === 'string' ? [...value].length : Array.isArray(value) ? value.length : undefined;
  if (length !== undefined) {
    const [min, max] = typeof value === 'string' ? ['minLength', 'maxLength'] : ['minItems', 'maxItems'];
    if (typeof s.raw[min] === 'number' && length < s.raw[min]) fail(path, `below ${min}`);
    if (typeof s.raw[max] === 'number' && length > s.raw[max]) fail(path, `above ${max}`);
  }
  if (Array.isArray(value)) {
    if (s.items) value.forEach((v, i) => validateJSON(v, s.items!, `${path}[${i}]`));
  } else if (typeof value === 'object') {
    for (const key of s.required) if (!hasOwn(value, key)) fail(path, `missing required field "${key}"`);
    for (const [key, v] of Object.entries(value)) {
      if (hasOwn(s.properties, key)) validateJSON(v, s.properties[key], `${path}.${key}`);
      else if (s.raw.additionalProperties === false) fail(path, `unknown field "${key}"`);
    }
  }
}

const ref = (name: string): AST => ({ _node_type: 'VarRef', name });
const lit = (val: string | number | boolean | null): AST => ({ _node_type: 'Literal', val });
const call = (name: string, ...args: AST[]): AST => ({ _node_type: 'FuncCall', name, args });
const block = (...stmts: AST[]): AST => ({ _node_type: 'Block', stmts });
const fn = (parameters: string[], ...stmts: AST[]): AST =>
  ({ _node_type: 'FunctionDefNode', parameters, body: block(...stmts) });
const when = (condition: AST, ...trueBranch: AST[]): AST =>
  ({ _node_type: 'IfNode', condition, trueBranch, falseBranch: [] });
const get = (value: AST, key: string): AST => call('getProp', value, lit(key));
const eq = (a: AST, b: AST): AST => call('equal', a, b);
const assign = (name: string, value: AST): AST => call('setq', ref(name), value);
const put = (map: AST, key: AST, value: AST): AST => call('setProp', map, key, value);
function data(value: JSONValue): AST {
  if (Array.isArray(value)) return call('array', ...value.map(data));
  if (value && typeof value === 'object') {
    return call('mapValue', ...Object.keys(value).sort().flatMap(k => [lit(k), data(value[k])]));
  }
  return lit(value);
}
function guard(condition: AST, message: string): AST {
  return when(call('not', condition), call('error', lit(`FTS ${message}`)));
}
function typedFunction(name: string, parameters: string[], contracts: string[], result: string, body: AST[]): AST {
  return assign(name, call('withSignature', fn(parameters, ...body),
    call('signature', call('array', ...contracts.map(lit)), lit(result))));
}

function checks(s: Schema, value: AST, path: string, counter: { n: number }): AST[] {
  const type = call('valueType', value);
  const matches = s.types.map(t => t === 'null' ? eq(value, lit(null)) : eq(type, lit(typeCodes[t])));
  const body: AST[] = [guard(call('or', ...matches), `${path}: expected ${s.types.join('|')}`)];
  if (s.raw.enum) body.push(guard(call('or', ...array(s.raw.enum, path).map(v => eq(value, data(v)))), `${path}: not in enum`));
  const numberChecks: AST[] = [];
  if (s.types.includes('integer') && !s.types.includes('number')) {
    numberChecks.push(guard(eq(call('floor', value), value), `${path}: expected integer`));
  }
  for (const [key, op] of [['minimum', 'biggerEq'], ['maximum', 'smallerEq']]) {
    if (typeof s.raw[key] === 'number') numberChecks.push(guard(call(op, value, data(s.raw[key])), `${path}: ${key}`));
  }
  if (numberChecks.length) body.push(when(eq(type, lit('N')), ...numberChecks));
  for (const [code, min, max] of [['S', 'minLength', 'maxLength'], ['A', 'minItems', 'maxItems']]) {
    const limits: AST[] = [];
    for (const [key, op] of [[min, 'biggerEq'], [max, 'smallerEq']]) {
      const collection = code === 'S' ? call('split', value, lit('')) : value;
      if (typeof s.raw[key] === 'number') limits.push(guard(call(op, call('length', collection), data(s.raw[key])), `${path}: ${key}`));
    }
    if (limits.length) body.push(when(eq(type, lit(code)), ...limits));
  }
  if (s.types.includes('object')) {
    const id = counter.n++;
    const seen = ref(`seen${id}`), key = ref(`key${id}`), item = ref(`item${id}`);
    const known = Object.keys(s.properties).sort();
    const callback: AST[] = [put(seen, key, lit(true))];
    for (const k of known) callback.push(when(eq(key, lit(k)), ...checks(s.properties[k], item, `${path}.${k}`, counter)));
    if (s.raw.additionalProperties === false) {
      callback.push(guard(known.length ? call('or', ...known.map(k => eq(key, lit(k)))) : lit(false), `${path}: unknown field`));
    }
    body.push(when(eq(type, lit('M')), assign(`seen${id}`, call('mapValue')),
      call('apply', fn([`key${id}`, `item${id}`], ...callback), value),
      ...s.required.map(k => guard(eq(get(seen, k), lit(true)), `${path}.${k}: required field`))));
  }
  if (s.items) {
    const id = counter.n++;
    body.push(when(eq(type, lit('A')),
      call('apply', fn([`index${id}`, `item${id}`], ...checks(s.items, ref(`item${id}`), `${path}[]`, counter)), value)));
  }
  return body;
}

function classAST(a: Archetype): AST[] {
  const fields = Object.keys(a.properties.properties).sort();
  for (const reserved of ['ftsState', 'ftsExtraAttributes']) {
    if (fields.includes(reserved)) fail(a.key, `field "${reserved}" is reserved by the importer`);
  }
  const declaration = call('class', ref(a.name), block(
    ...fields.map(k => {
      const s = a.properties.properties[k];
      const required = a.properties.required.includes(k);
      const hasDefault = hasOwn(a.defaults, k);
      const contract = s.types.length === 1 && s.types[0] !== 'null' && (required || hasDefault) ? typeCodes[s.types[0]] : 'V';
      return call('field', lit(k), lit(contract), ...hasDefault ? [data(a.defaults[k])] : required ? [] : [lit(null)]);
    }),
    call('field', lit('ftsState'), lit('M'), data(a.stateDefaults)),
    call('field', lit('ftsExtraAttributes'), lit('M'), call('mapValue')),
  ));
  const defaults = [
    assign('attrs', data(a.defaults)), assign('stateValues', data(a.stateDefaults)),
    call('apply', fn(['key', 'value'], put(ref('attrs'), ref('key'), ref('value'))), ref('attributes')),
    call('apply', fn(['key', 'value'], put(ref('stateValues'), ref('key'), ref('value'))), ref('state')),
  ];
  const validation = [
    ...checks(a.properties, ref('attrs'), `${a.name}.attributes`, { n: 0 }),
    ...checks(a.state, ref('stateValues'), `${a.name}.state`, { n: 10000 }),
  ];
  const known = fields.length ? call('or', ...fields.map(k => eq(ref('key'), lit(k)))) : lit(false);
  return [
    declaration,
    typedFunction(`validate${a.name}`, ['attributes', 'state'], ['M', 'M'], 'L',
      [...defaults, ...validation, lit(true)]),
    typedFunction(`create${a.name}`, ['attributes', 'state'], ['M', 'M'], 'O', [
      call(`validate${a.name}`, ref('attributes'), ref('state')),
      ...defaults,
      assign('values', call('mapValue', lit('ftsState'), ref('stateValues'))),
      assign('extras', call('mapValue')),
      call('apply', fn(['key', 'value'], {
        _node_type: 'IfNode', condition: known,
        trueBranch: [put(ref('values'), ref('key'), ref('value'))],
        falseBranch: [put(ref('extras'), ref('key'), ref('value'))],
      }), ref('attrs')),
      put(ref('values'), lit('ftsExtraAttributes'), ref('extras')),
      call('new', ref(a.name), ref('values')),
    ]),
  ];
}

/** Convert a composed FTS modelseed to self-contained executable Chariot source. */
export function importFTSModel(input: unknown): FTSImportResult {
  const model = object(json(input), '$');
  if (array(model.imports ?? [], 'imports').length) fail('imports', 'compose imports with loadFTSModel before conversion');
  const descriptor = object(model.descriptor, 'descriptor');
  const root = text(descriptor.rootFrameId, 'descriptor.rootFrameId');
  const prefix = text(descriptor.anchorPrefix, 'descriptor.anchorPrefix');
  const frames = unique(entries(model, 'frames'), 'alias', 'frames');
  const canonical = unique([...frames.values()], 'canonicalId', 'frames');
  if (!canonical.has(root)) fail('descriptor.rootFrameId', 'root frame not found');
  const schemaFrames = [...frames.values()].filter(f => f.canonicalId === root || text(f.canonicalId, 'canonicalId').startsWith(prefix));
  const schemaAliases = new Set(schemaFrames.map(f => text(f.alias, 'alias')));
  const anchors = schemaFrames.filter(f => f.canonicalId !== root);
  const anchorKeys = unique(anchors, 'key', 'anchors');
  const archetypes = [...unique(entries(model, 'archetypes'), 'key', 'archetypes').values()].map(raw => {
    const key = text(raw.key, 'archetype.key');
    const parent = text(raw.parentNeighborhoodKey, `${key}.parentNeighborhoodKey`);
    const anchor = anchorKeys.get(parent);
    if (!anchor) fail(key, `unknown parent neighborhood "${parent}"`);
    const stem = key.startsWith('instance.') ? key.slice('instance.'.length) : key;
    const parts = stem.split('.').filter(p => !['entity', 'record'].includes(p));
    const name = parts.join('_').split(/[^A-Za-z0-9]+/).filter(Boolean).map(p => p[0].toUpperCase() + p.slice(1)).join('');
    if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(name)) fail(key, 'cannot form a Chariot class name');
    const properties = schema(raw.propertySchema ?? { type: 'object' }, `${key}.propertySchema`);
    const state = schema(raw.stateSchema ?? { type: 'object' }, `${key}.stateSchema`);
    if (properties.types.join() !== 'object' || state.types.join() !== 'object') fail(key, 'attribute and state schemas must be objects');
    const defaults = object(raw.defaultAttributes ?? {}, `${key}.defaultAttributes`);
    const stateDefaults = object(raw.defaultState ?? {}, `${key}.defaultState`);
    validateJSON(defaults, { ...properties, required: [] }, `${key}.defaultAttributes`);
    validateJSON(stateDefaults, { ...state, required: [] }, `${key}.defaultState`);
    if (Object.keys(object(raw.validationRules ?? {}, `${key}.validationRules`)).length) fail(key, 'validationRules are not supported');
    return { raw, key, name, parent, properties, state, defaults, stateDefaults };
  }).sort((a, b) => a.key.localeCompare(b.key, 'en'));
  if (!archetypes.length) fail('archetypes', 'no business archetypes found');
  const symbols = new Set(['ftsModel']);
  for (const a of archetypes) {
    for (const name of [a.name, `create${a.name}`, `validate${a.name}`]) {
      if (symbols.has(name)) fail(a.key, `generated name "${name}" collides`);
      symbols.add(name);
    }
  }
  const edges = [...unique(entries(model, 'edges'), 'key', 'edges').values()].sort((a, b) =>
    text(a.key, 'edge.key').localeCompare(text(b.key, 'edge.key'), 'en'));
  const relationships: JSONObject[] = [], hierarchy: JSONObject[] = [];
  let excludedInstanceEdges = 0;
  for (const edge of edges) {
    const from = text(edge.fromAlias, 'edge.fromAlias'), to = text(edge.toAlias, 'edge.toAlias');
    if (!frames.has(from) || !frames.has(to)) fail(String(edge.key), 'unresolved edge endpoint');
    text(edge.typeName, `${String(edge.key)}.typeName`);
    if (!schemaAliases.has(from) || !schemaAliases.has(to)) { excludedInstanceEdges++; continue; }
    if (frames.get(from)!.canonicalId === root || frames.get(to)!.canonicalId === root) {
      hierarchy.push(edge);
      continue;
    }
    const mysql = edge['x-relational'] === undefined ? {} :
      object(object(edge['x-relational'], 'x-relational').mysql ?? {}, 'x-relational.mysql');
    const projection = object(mysql.associationTable ?? mysql.joinTable ?? mysql.foreignKey ?? {}, 'edge projection');
    function endpoint(alias: string, explicit: JSONValue | undefined): Archetype {
      const parent = frames.get(alias)!.key;
      const candidates = archetypes.filter(a => a.parent === parent && (explicit === undefined || a.key === explicit));
      if (candidates.length !== 1) fail(String(edge.key), `ambiguous or missing archetype for endpoint "${alias}"`);
      return candidates[0];
    }
    const source = endpoint(from, projection.sourceArchetypeKey);
    const target = endpoint(to, projection.targetArchetypeKey);
    relationships.push({ ...edge, sourceClass: source.name, targetClass: target.name,
      sourceArchetypeKey: source.key, targetArchetypeKey: target.key });
  }
  const metadata: JSONObject = {
    name: model.name ?? '', descriptor, catalog: model.catalog ?? {},
    metadata: model.metadata ?? {}, sourceNotes: model.sourceNotes ?? [],
    frames: schemaFrames, hierarchy,
    classes: archetypes.map(a => ({ ...a.raw, className: a.name })),
    relationships,
    excludedInstanceFrames: frames.size - schemaFrames.length, excludedInstanceEdges,
  };
  if (model['x-relational'] !== undefined) metadata['x-relational'] = model['x-relational'];
  const ast = block(...archetypes.flatMap(classAST), assign('ftsModel', data(metadata)));
  return {
    ast, source: `// Generated from FTS modelseed. Use create<Class>(attributes, state) for checked construction.\n${generateChariotSourceFromAST(ast)}\n`,
    classNames: archetypes.map(a => a.name),
    relationshipKeys: relationships.map(e => text(e.key, 'relationship.key')),
    excludedInstanceFrames: frames.size - schemaFrames.length, excludedInstanceEdges,
  };
}
