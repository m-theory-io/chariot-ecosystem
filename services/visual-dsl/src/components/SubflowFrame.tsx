import React from 'react';
import { useNesting } from '../contexts/NestingContext';

type Bounds = { x: number; y: number; width: number; height: number };

export default function SubflowFrame({ groupId, bounds }: { groupId: string; title?: string; bounds: Bounds }) {
  // We keep the state access in case of future extension but do not render controls now
  const { getSubflow } = useNesting();
  void getSubflow(groupId);

  return (
    <div
      style={{
        position: 'absolute',
        left: bounds.x,
        top: bounds.y,
        width: bounds.width,
        height: bounds.height,
  borderRadius: 16,
        borderWidth: 2,
        borderStyle: 'dashed',
        borderColor: '#8b5cf6', // purple dashed border
        background: 'transparent',
        boxShadow: 'none',
        pointerEvents: 'none',
        zIndex: 10,
      }}
    />
  );
}
