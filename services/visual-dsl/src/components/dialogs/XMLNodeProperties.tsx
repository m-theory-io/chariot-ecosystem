import React, { useEffect, useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface XMLNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: XMLNodeProperties) => void;
  onDelete: () => void;
  initialProperties: XMLNodeProperties;
}

// Backend semantics:
// - No args: NewXMLNode("xml")
// - One string arg: parse as XML content and return the root
export interface XMLNodeProperties {
  xmlString?: string;
}

export const XMLNodePropertiesDialog: React.FC<XMLNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties
}) => {
  const [xmlString, setXmlString] = useState(initialProperties.xmlString ?? '');
  const [canSave, setCanSave] = useState(true); // optional field

  useEffect(() => {
    setCanSave(true);
  }, [xmlString]);

  const handleSave = () => {
    if (!canSave) return;
    const trimmed = xmlString.trim();
    onSave(trimmed ? { xmlString: trimmed } : {});
    onClose();
  };

  const handleClose = () => {
    const trimmed = xmlString.trim();
    onSave(trimmed ? { xmlString: trimmed } : {});
    onClose();
  };



  const handleCancel = () => {
    onClose();
  };
  const handleDelete = () => {
    onDelete();
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-md w-full mx-4">
        {/* Title Bar */}
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">XML Node Properties</h3>
          <button
            onClick={handleCancel}
            className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200"
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div className="p-6 space-y-6">
          {/* Optional XML content input */}
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">XML String (optional):</label>
            <Input type="text" value={xmlString} onChange={(e) => setXmlString(e.target.value)} className="w-full" placeholder="<root><item/></root>" />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">Leave blank to create an empty XML node named "xml". If provided, the string will be parsed as XML.</p>
          </div>

          {/* Buttons */}
          <div className="flex gap-3">
            <Button onClick={handleSave} disabled={!canSave} className={`px-6 py-2 ${canSave ? 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600' : 'bg-gray-200 dark:bg-gray-600 opacity-60 cursor-not-allowed'} text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200`}>Save Properties</Button>
            <Button onClick={handleDelete} className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200">Delete</Button>
          </div>
        </div>
      </div>

      {/* Help */}
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">Creates an XML node. If XML content is provided, it’s parsed and the root element becomes the node.</p>
        <ul className="text-xs text-gray-600 dark:text-gray-400 mt-1 space-y-1">
          <li>1. xmlString (string, optional)</li>
        </ul>
      </div>
    </div>
  );
};
