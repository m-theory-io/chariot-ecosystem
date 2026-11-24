import React, { useEffect, useId, useRef, useState } from 'react';
import {
  CLASS_CONTRACT_CODES, CLASS_NODE_SCHEMA, getClassNodeDefaults, validateClassNodeProperties,
  type ClassNodeLabel, type ClassFunctionParameter, type ClassValidationIssue,
} from 'chariot-codegen';
import type { ClassProperties } from '../../utils/classAuthoring';

interface Props {
  nodeLabel: ClassNodeLabel;
  initialProperties?: ClassProperties;
  onSave: (properties: ClassProperties) => void;
  onClose: () => void;
  onDelete: () => void;
  returnFocusNodeId?: string;
  validate?: (properties: ClassProperties) => ClassValidationIssue[];
}

const titles: Record<string, string> = {
  identifier: 'Declaration identifier', memberName: 'Member name', contract: 'Contract',
  hasDefault: 'Default present', defaultExpression: 'Default expression', classExpression: 'Class expression',
  functionMode: 'Function mode', parameters: 'Parameters', body: 'Body', functionExpression: 'Function expression',
  hasSignature: 'Include method signature', signatureExpression: 'Signature expression',
  hasAttributes: 'Attributes present', attributesExpression: 'Attributes expression',
  parameterContracts: 'Parameter contracts', returnContract: 'Return contract',
  valueExpression: 'Value expression', interfaceExpression: 'Interface expression',
};
const control = 'w-full rounded border border-gray-500 bg-white dark:bg-gray-800 p-2 text-sm';

/** Schema-driven, but fields retain the canonical construct-specific contracts. */
export function ClassNodePropertiesDialog({ nodeLabel, initialProperties, onSave, onClose, onDelete, validate, returnFocusNodeId }: Props) {
  const [draft, setDraft] = useState<Record<string, unknown>>(() =>
    structuredClone(initialProperties ?? getClassNodeDefaults(nodeLabel)) as unknown as Record<string, unknown>);
  const [attempted, setAttempted] = useState(false);
  const dialog = useRef<HTMLDivElement>(null);
  const id = useId();
  const schema = CLASS_NODE_SCHEMA[nodeLabel];
  const issues = [...validateClassNodeProperties(nodeLabel, draft), ...(validate?.(draft as unknown as ClassProperties) ?? [])];
  const update = (key: string, value: unknown) => setDraft(previous => ({ ...previous, [key]: value }));

  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    dialog.current?.querySelector<HTMLElement>('input, select, textarea, button')?.focus();
    return () => {
      if (previous?.isConnected) previous.focus();
      else if (returnFocusNodeId) Array.from(document.querySelectorAll<HTMLElement>('.react-flow__node'))
        .find(node => node.dataset.id === returnFocusNodeId)?.focus();
    };
  }, []);

  const contractSelect = (value: unknown, change: (value: string) => void, label: string, fieldId?: string) => (
    <select id={fieldId} aria-label={label} className={control} value={String(value ?? '')} onChange={event => change(event.target.value)}>
      {!CLASS_CONTRACT_CODES.includes(value as typeof CLASS_CONTRACT_CODES[number]) && <option value={String(value ?? '')}>Choose contract</option>}
      {CLASS_CONTRACT_CODES.map(code => <option key={code} value={code}>{code}</option>)}
    </select>
  );

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4" onMouseDown={event => event.stopPropagation()}>
      <div ref={dialog} role="dialog" aria-modal="true" aria-labelledby={`${id}-title`} aria-describedby={`${id}-help`}
        className="bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 rounded-lg border shadow-xl w-full max-w-3xl max-h-[90vh] overflow-y-auto p-6"
        onKeyDown={event => {
          event.stopPropagation();
          if (event.key === 'Escape') { event.preventDefault(); onClose(); }
          if (event.key === 'Tab') {
            const controls = Array.from(dialog.current?.querySelectorAll<HTMLElement>('button:not(:disabled), input, select, textarea') ?? []);
            const first = controls[0], last = controls[controls.length - 1];
            if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
            else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
          }
        }}>
        <h2 id={`${id}-title`} className="text-lg font-semibold">{nodeLabel} Properties</h2>
        <p id={`${id}-help`} className="text-sm my-3">{schema.help} Expressions are raw Chariot; dynamic conformance is checked at runtime.</p>
        {schema.bodyChildLabel && <p className="text-sm mb-3">Ordered body: enable Nesting Mode, select this {nodeLabel}, then add {schema.bodyChildLabel} nodes in order. Methods are registered outside declarations.</p>}
        <form className="space-y-4" onSubmit={event => {
          event.preventDefault();
          setAttempted(true);
          if (issues.length) {
            dialog.current?.querySelector<HTMLElement>('[aria-invalid="true"]')?.focus();
            return;
          }
          onSave(structuredClone(draft) as unknown as ClassProperties);
          onClose();
        }}>
          {Object.entries(schema.properties).map(([key, field]) => {
            if (field.activeWhen && draft[field.activeWhen.property] !== field.activeWhen.equals) return null;
            const fieldId = `${id}-${key}`;
            const errors = issues.filter(issue => issue.property === key || issue.property.startsWith(`${key}.`));
            const label = titles[key] ?? key;
            const value = draft[key];
            return <div key={key}>
              <label htmlFor={fieldId} className="block text-sm font-medium mb-1">{label}</label>
              {field.kind === 'toggle' ? <input id={fieldId} type="checkbox" checked={value === true} onChange={event => update(key, event.target.checked)} />
                : field.kind === 'contract' ? contractSelect(value, next => update(key, next), label, fieldId)
                : field.kind === 'functionMode' ? <select id={fieldId} className={control} value={String(value ?? '')} onChange={event => update(key, event.target.value)}>
                  <option value="editor">Parameter / body editor</option><option value="expression">Function expression</option>
                </select>
                : field.kind === 'parameters' ? <fieldset id={fieldId} className="space-y-2">
                  {(Array.isArray(value) ? value : []).map((row: ClassFunctionParameter, index) => <div key={index} className="flex gap-2">
                    {(['name', 'value'] as const).map(part => <input key={part} className={control} aria-label={`Parameter ${index + 1} ${part}`} value={row?.[part] ?? ''}
                      placeholder={part === 'name' ? 'name' : 'Default expression (optional)'}
                      onChange={event => update(key, (value as ClassFunctionParameter[]).map((entry, i) => i === index ? { ...entry, [part]: event.target.value } : entry))} />)}
                    <button type="button" aria-label={`Remove parameter ${index + 1}`} onClick={() => update(key, (value as ClassFunctionParameter[]).filter((_, i) => i !== index))}>Remove</button>
                  </div>)}
                  <button type="button" onClick={() => update(key, [...(Array.isArray(value) ? value : []), { name: '', value: '' }])}>Add Parameter</button>
                  <p className="text-xs">First parameter must be self without a default. Other values are optional default expressions, matching the Function editor.</p>
                </fieldset>
                : field.kind === 'contracts' ? <fieldset id={fieldId} className="space-y-2">
                  {(Array.isArray(value) ? value : []).map((code, index) => <div key={index} className="flex gap-2">
                    {contractSelect(code, next => update(key, (value as string[]).map((entry, i) => i === index ? next : entry)), `Parameter contract ${index + 1}`)}
                    <button type="button" aria-label={`Remove contract ${index + 1}`} onClick={() => update(key, (value as string[]).filter((_, i) => i !== index))}>Remove</button>
                  </div>)}
                  <button type="button" onClick={() => update(key, [...(Array.isArray(value) ? value : []), 'V'])}>Add Contract</button>
                  <p className="text-xs">Rows are emitted in order. No rows means zero parameters.</p>
                </fieldset>
                : field.kind === 'body' || field.kind === 'expression' ? <textarea id={fieldId} className={`${control} font-mono ${field.kind === 'body' ? 'h-80' : 'min-h-20'}`} value={typeof value === 'string' ? value : ''}
                  aria-invalid={errors.length > 0} aria-describedby={errors.length ? `${fieldId}-errors` : undefined} onChange={event => update(key, event.target.value)} />
                : <input id={fieldId} className={control} value={typeof value === 'string' ? value : ''} aria-invalid={errors.length > 0}
                  aria-describedby={errors.length ? `${fieldId}-errors` : undefined} onChange={event => update(key, event.target.value)} />}
              {!!errors.length && <ul id={`${fieldId}-errors`} className="text-sm text-red-600" role="alert">{errors.map((issue, index) => <li key={index}>{issue.property}: {issue.message}</li>)}</ul>}
            </div>;
          })}
          {issues.some(issue => !(issue.property.split('.')[0] in schema.properties)) && <ul role="alert" className="text-red-600 text-sm">{issues.filter(issue => !(issue.property.split('.')[0] in schema.properties)).map((issue, index) => <li key={index}>{issue.message} {declarationHint(nodeLabel)}</li>)}</ul>}
          {attempted && !!issues.length && <p role="alert">Fix the diagnostics before saving. Your edits have been retained.</p>}
          <div className="flex gap-4 border-t pt-4">
            <button type="submit" className="rounded bg-blue-600 text-white px-4 py-2">Save Properties</button>
            <button type="button" onClick={onClose}>Cancel</button>
            <button type="button" className="ml-auto text-red-600" onClick={() => { onDelete(); onClose(); }}>Delete</button>
          </div>
        </form>
      </div>
    </div>
  );
}

function declarationHint(label: ClassNodeLabel) {
  return label === 'Field' || label === 'Requires' ? 'Cancel and enable Nesting Mode on the matching declaration, or drag its connector to this node.' : '';
}