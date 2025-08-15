import React, { createContext, useContext, useState } from 'react';

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
  };

  const getChildrenOf = (parentId: string) => {
    return nestingRelations
      .filter(rel => rel.parentId === parentId)
      .sort((a, b) => a.order - b.order);
  };

  const getParentOf = (childId: string) => {
    return nestingRelations.find(rel => rel.childId === childId) || null;
  };

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
      getParentOf
    }}>
      {children}
    </NestingContext.Provider>
  );
};
