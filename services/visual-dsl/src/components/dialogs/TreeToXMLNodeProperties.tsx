import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface TreeToXMLNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: TreeToXMLNodeProperties) => void;
  onDelete: () => void;
  initialProperties: TreeToXMLNodeProperties;
}

export interface TreeToXMLNodeProperties {
  treeVariable: string;
  prettyPrint?: boolean;
}

export const TreeToXMLNodePropertiesDialog: React.FC<TreeToXMLNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties,
}) => {
  const [treeVariable, setTreeVariable] = useState(initialProperties.treeVariable || 'tree');
  const [prettyPrint, setPrettyPrint] = useState<boolean>(
    initialProperties.prettyPrint !== undefined ? Boolean(initialProperties.prettyPrint) : true
  );

  const commit = () => {
    onSave({ treeVariable: treeVariable.trim(), prettyPrint });
  };

  const handleSave = () => { commit(); onClose(); };
  const handleClose = () => { commit(); onClose(); };


  const handleCancel = () => {
    onClose();
  };
  const handleDelete = () => { onDelete(); onClose(); };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Tree To XML Properties</h3>
          <button onClick={handleCancel} className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200">×</button>
        </div>
        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tree Variable:</label>
            <Input type="text" value={treeVariable} onChange={(e) => setTreeVariable(e.target.value)} placeholder="tree" />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Converts the specified TreeNode into XML text.</p>
          </div>
          <div className="mt-2">
            <label className="flex items-center space-x-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              <input type="checkbox" checked={prettyPrint} onChange={(e) => setPrettyPrint(e.target.checked)} className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 dark:focus:ring-blue-400 dark:bg-gray-700" />
              <span>Pretty print</span>
            </label>
          </div>
          <div className="flex gap-3 pt-2">
            <Button onClick={handleClose} className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200">Save Properties</Button>
            <Button onClick={handleDelete} className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200">Delete</Button>
          </div>
        </div>
      </div>
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">Returns XML string for the given tree. Toggle pretty print for formatted output.</p>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">💡 Example:<br/>setq(xml, treeToXML(tree, true))</p>
      </div>
    </div>
  );
};

export default TreeToXMLNodePropertiesDialog;
