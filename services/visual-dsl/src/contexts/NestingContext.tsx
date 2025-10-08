import React, { createContext, useContext, useMemo, useState, useCallback } from 'react';

export interface Subflow {
  id: string;
  name: string;
  inputPorts: { id: string; label: string }[];
  outputPorts: { id: string; label: string }[];
  collapsed: boolean;
}

interface NestingRelation {
  parentId: string;
  childId: string;
  order: number; // Parameter order in function call
}

interface NestingContextType {
  nestingMode: boolean;
  setNestingMode: (mode: boolean) => void;
  selectedParentId: string | null;
  setSelectedParentId: (parentId: string | null) => void;
  nestingRelations: NestingRelation[];
  addNestingRelation: (relation: NestingRelation) => void;
  removeNestingRelation: (parentId: string, childId: string) => void;
  getChildrenOf: (parentId: string) => NestingRelation[];
  getParentOf: (childId: string) => NestingRelation | null;
  // Subflow APIs
  wrapGroupAsSubflow: (groupId: string, opts?: { name?: string; inputs?: { id: string; label: string }[]; outputs?: { id: string; label: string }[] }) => void;
  unwrapSubflow: (groupId: string) => void;
  toggleSubflowCollapsed: (groupId: string, collapsed?: boolean) => void;
  openSubflow: (groupId: string) => void;
  closeSubflow: () => void;
  isSubflow: (groupId: string) => boolean;
  getSubflow: (groupId: string) => Subflow | undefined;
  getActiveSubflow: () => { subflowId: string; title: string } | undefined;
  // Persistence helpers for subflows
  getAllSubflows: () => Record<string, Subflow>;
  replaceAllSubflows: (map: Record<string, Subflow>) => void;
}

const NestingContext = createContext<NestingContextType | undefined>(undefined);

export const useNesting = () => {
  const context = useContext(NestingContext);
  if (context === undefined) {
    throw new Error('useNesting must be used within a NestingProvider');
  }
  return context;
};

export const NestingProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [nestingMode, setNestingMode] = useState(false);
  const [selectedParentId, setSelectedParentId] = useState<string | null>(null);
  const [nestingRelations, setNestingRelations] = useState<NestingRelation[]>([]);
  const [subflows, setSubflows] = useState<Record<string, Subflow>>({});
  const [subflowNavStack, setSubflowNavStack] = useState<{ subflowId: string; title: string }[]>([]);

  // Derive a consistent group/subflow id for a given parent id
  const groupIdForParent = useCallback((parentId: string) => `group-${parentId}`, []);

  // Ensure a subflow exists for a given parent (id may be parent or group id)
  const ensureSubflowForParent = useCallback((parentIdOrGroup: string, opts?: { name?: string; inputs?: { id: string; label: string }[]; outputs?: { id: string; label: string }[] }) => {
    const parentId = parentIdOrGroup.startsWith('group-') ? parentIdOrGroup.substring(6) : parentIdOrGroup;
    const gid = groupIdForParent(parentId);
    setSubflows(prev => {
      const existing = prev[gid];
      const next = existing ?? {
        id: gid,
        name: `Subflow ${parentId}`,
        inputPorts: [{ id: 'in', label: 'In' }],
        outputPorts: [{ id: 'out', label: 'Out' }],
        collapsed: false,
      };
      const withOpts = {
        ...next,
        name: opts?.name ?? next.name,
        inputPorts: opts?.inputs ?? next.inputPorts,
        outputPorts: opts?.outputs ?? next.outputPorts,
      };
      if (existing && existing === withOpts) return prev; // no change
      return { ...prev, [gid]: withOpts };
    });
  }, [groupIdForParent]);

  // Remove a subflow for a given parent (id may be parent or group id)
  const removeSubflowForParent = useCallback((parentIdOrGroup: string) => {
    const parentId = parentIdOrGroup.startsWith('group-') ? parentIdOrGroup.substring(6) : parentIdOrGroup;
    const gid = groupIdForParent(parentId);
    setSubflows(prev => {
      if (!prev[gid]) return prev;
      const { [gid]: _, ...rest } = prev;
      return rest;
    });
    setSubflowNavStack(prev => prev.filter(e => e.subflowId !== gid));
  }, [groupIdForParent]);

  const addNestingRelation = (relation: NestingRelation) => {
    console.log('Adding nesting relation:', relation);
    setNestingRelations(prev => {
      const newRelations = [...prev, relation];
      console.log('Updated nesting relations:', newRelations);
      return newRelations;
    });
  };

  const removeNestingRelation = (parentId: string, childId: string) => {
    setNestingRelations(prev => 
      prev.filter(rel => !(rel.parentId === parentId && rel.childId === childId))
    );
    // Cleanup is handled by the effect syncing subflows below
  };

  const getChildrenOf = (parentId: string) => {
    return nestingRelations
      .filter(rel => rel.parentId === parentId)
      .sort((a, b) => a.order - b.order);
  };

  const getParentOf = (childId: string) => {
    return nestingRelations.find(rel => rel.childId === childId) || null;
  };

  // Subflow helpers
  const wrapGroupAsSubflow = useCallback((groupId: string, opts?: { name?: string; inputs?: { id: string; label: string }[]; outputs?: { id: string; label: string }[] }) => {
    ensureSubflowForParent(groupId, opts);
  }, [ensureSubflowForParent]);

  const unwrapSubflow = useCallback((groupId: string) => {
    removeSubflowForParent(groupId);
  }, [removeSubflowForParent]);

  const toggleSubflowCollapsed = useCallback((groupId: string, collapsed?: boolean) => {
    setSubflows(prev => {
      const sf = prev[groupId];
      if (!sf) return prev;
      const next = { ...sf, collapsed: collapsed ?? !sf.collapsed };
      return { ...prev, [groupId]: next };
    });
  }, []);

  const openSubflow = useCallback((groupId: string) => {
    setSubflowNavStack(prev => [...prev, { subflowId: groupId, title: subflows[groupId]?.name ?? 'Subflow' }]);
  }, [subflows]);

  const closeSubflow = useCallback(() => {
    setSubflowNavStack(prev => prev.slice(0, -1));
  }, []);

  const isSubflow = useCallback((groupId: string) => Boolean(subflows[groupId]), [subflows]);
  const getSubflow = useCallback((groupId: string) => subflows[groupId], [subflows]);
  const getActiveSubflow = useCallback(() => subflowNavStack[subflowNavStack.length - 1], [subflowNavStack]);

  // Persistence helpers
  const getAllSubflows = useCallback(() => subflows, [subflows]);
  const replaceAllSubflows = useCallback((map: Record<string, Subflow>) => {
    setSubflows(map || {});
  }, []);

  // Note: Subflows are managed manually (via wrap/unwrap) to avoid unexpected automatic grouping

  return (
    <NestingContext.Provider value={{ 
      nestingMode,
      setNestingMode,
      selectedParentId,
      setSelectedParentId,
      nestingRelations,
      addNestingRelation,
      removeNestingRelation,
      getChildrenOf,
      getParentOf,
      wrapGroupAsSubflow,
      unwrapSubflow,
      toggleSubflowCollapsed,
      openSubflow,
      closeSubflow,
      isSubflow: useCallback((groupId: string) => {
        const gid = groupId.startsWith('group-') ? groupId : groupIdForParent(groupId);
        return Boolean(subflows[gid]);
      }, [subflows, groupIdForParent]),
      getSubflow: useCallback((groupId: string) => {
        const gid = groupId.startsWith('group-') ? groupId : groupIdForParent(groupId);
        return subflows[gid];
      }, [subflows, groupIdForParent]),
      getActiveSubflow,
      getAllSubflows,
      replaceAllSubflows,
    }}>
      {children}
    </NestingContext.Provider>
  );
};
