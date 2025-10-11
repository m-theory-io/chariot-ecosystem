import React, { useState } from 'react';
import { Button } from '../ui/button';
import { Input } from '../ui/input';

interface TreeSaveSecureNodePropertiesProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (properties: TreeSaveSecureNodeProperties) => void;
  onDelete: () => void;
  initialProperties: TreeSaveSecureNodeProperties;
}

export interface TreeSaveSecureNodeProperties {
  treeVariable: string;
  filename: string;
  encryptionKeyID: string;
  signingKeyID: string;
  watermark: string;
  verificationKeyID?: string;
  checksum?: boolean;
  auditTrail?: boolean;
  compressionLevel?: number;
}

export const TreeSaveSecureNodePropertiesDialog: React.FC<TreeSaveSecureNodePropertiesProps> = ({
  isOpen,
  onClose,
  onSave,
  onDelete,
  initialProperties,
}) => {
  const [treeVariable, setTreeVariable] = useState(initialProperties.treeVariable || 'tree');
  const [filename, setFilename] = useState(initialProperties.filename || 'secure.json');
  const [encryptionKeyID, setEncryptionKeyID] = useState(initialProperties.encryptionKeyID || 'encKey');
  const [signingKeyID, setSigningKeyID] = useState(initialProperties.signingKeyID || 'signKey');
  const [watermark, setWatermark] = useState(initialProperties.watermark || 'watermark');
  const [verificationKeyID, setVerificationKeyID] = useState(initialProperties.verificationKeyID || '');
  const [checksum, setChecksum] = useState(initialProperties.checksum !== undefined ? Boolean(initialProperties.checksum) : true);
  const [auditTrail, setAuditTrail] = useState(initialProperties.auditTrail !== undefined ? Boolean(initialProperties.auditTrail) : true);
  const [compressionLevel, setCompressionLevel] = useState<string>(
    typeof initialProperties.compressionLevel === 'number'
      ? String(initialProperties.compressionLevel)
      : '9'
  );

  const commit = () => {
    onSave({
      treeVariable: treeVariable.trim(),
      filename: filename.trim(),
      encryptionKeyID: encryptionKeyID.trim(),
      signingKeyID: signingKeyID.trim(),
      watermark: watermark.trim(),
      verificationKeyID: verificationKeyID.trim() || undefined,
      checksum,
      auditTrail,
      compressionLevel: compressionLevel !== '' ? Number(compressionLevel) : undefined,
    });
  };

  const handleSave = () => { commit(); onClose(); };
  const handleClose = () => { commit(); onClose(); };
  const handleDelete = () => { onDelete(); onClose(); };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg border-2 border-gray-800 dark:border-gray-200 shadow-xl max-w-lg w-full mx-4">
        <div className="bg-gray-100 dark:bg-gray-700 px-4 py-2 border-b border-gray-800 dark:border-gray-200 flex justify-between items-center">
          <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100">Tree Save Secure Properties</h3>
          <button onClick={handleClose} className="text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200 text-xl font-bold w-6 h-6 flex items-center justify-center border border-gray-800 dark:border-gray-200">×</button>
        </div>
        <div className="p-6 space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Tree Variable:</label>
            <Input type="text" value={treeVariable} onChange={(e) => setTreeVariable(e.target.value)} placeholder="agent" />
            <p className="text-xs text-gray-500 dark:text-gray-400 mt-1">The TreeNode variable to save securely</p>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Filename:</label>
            <Input type="text" value={filename} onChange={(e) => setFilename(e.target.value)} placeholder="secure.json" />
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Encryption Key ID:</label>
              <Input type="text" value={encryptionKeyID} onChange={(e) => setEncryptionKeyID(e.target.value)} placeholder="encKey" />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Signing Key ID:</label>
              <Input type="text" value={signingKeyID} onChange={(e) => setSigningKeyID(e.target.value)} placeholder="signKey" />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Watermark:</label>
            <Input type="text" value={watermark} onChange={(e) => setWatermark(e.target.value)} placeholder="confidential" />
          </div>
          <div className="pt-2 border-t border-gray-200 dark:border-gray-700">
            <p className="text-sm font-medium text-gray-800 dark:text-gray-200 mb-2">Advanced Options (optional)</p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Verification Key ID:</label>
                <Input type="text" value={verificationKeyID} onChange={(e) => setVerificationKeyID(e.target.value)} placeholder="verifyKey (defaults to signingKeyID)" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Compression Level (0-9):</label>
                <input
                  type="number"
                  min={0}
                  max={9}
                  value={compressionLevel}
                  onChange={(e) => setCompressionLevel(e.target.value)}
                  className="w-full px-3 py-2 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-md text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 dark:focus:ring-blue-400"
                />
              </div>
            </div>
            <div className="mt-2 space-y-2">
              <label className="flex items-center space-x-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                <input type="checkbox" checked={checksum} onChange={(e) => setChecksum(e.target.checked)} className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 dark:focus:ring-blue-400 dark:bg-gray-700" />
                <span>Enable checksum</span>
              </label>
              <label className="flex items-center space-x-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                <input type="checkbox" checked={auditTrail} onChange={(e) => setAuditTrail(e.target.checked)} className="rounded border-gray-300 dark:border-gray-600 text-blue-600 focus:ring-blue-500 dark:focus:ring-blue-400 dark:bg-gray-700" />
                <span>Enable audit trail</span>
              </label>
            </div>
          </div>
          <div className="flex gap-3 pt-2">
            <Button onClick={handleClose} className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200">Save Properties</Button>
            <Button onClick={handleDelete} className="px-6 py-2 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600 text-gray-800 dark:text-gray-200 border border-gray-800 dark:border-gray-200">Delete</Button>
          </div>
        </div>
      </div>
      <div className="absolute top-1/2 left-1/2 transform translate-x-64 -translate-y-1/2 max-w-sm p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-700 rounded-lg shadow-lg">
        <p className="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">Save a tree with encryption and signature. Optional options map controls verification key, checksum, audit trail, and compression.</p>
        <p className="text-xs text-blue-600 dark:text-blue-400 mt-2 italic">💡 Example:<br/>treeSaveSecure(agent, 'secure.json', 'encKey', 'signKey', 'watermark', map('checksum', true, 'compressionLevel', 9))</p>
      </div>
    </div>
  );
};
