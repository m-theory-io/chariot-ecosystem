import React, { useState } from 'react';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { ChariotCodeGenerator, type GenerateOptions } from 'chariot-codegen';

interface ChariotCodeGeneratorProps {
  diagramData: any;
}

export const ChariotCodeGeneratorComponent: React.FC<ChariotCodeGeneratorProps> = ({ diagramData }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [generatedCode, setGeneratedCode] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [error, setError] = useState('');

  const handleGenerate = async () => {
    if (!diagramData) {
      setError('No diagram data available');
      return;
    }

    setIsGenerating(true);
    setError('');

    try {
      const generator = new ChariotCodeGenerator(diagramData);
      const opts: GenerateOptions = { embedSource: false };
      const code = generator.generateChariotCode(opts);
      setGeneratedCode(code);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate code');
      console.error('Code generation error:', err);
    } finally {
      setIsGenerating(false);
    }
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(generatedCode);
      // Could add a toast notification here
    } catch (err) {
      console.error('Failed to copy to clipboard:', err);
    }
  };

  const handleDownload = () => {
    const filename = `${diagramData?.name || 'diagram'}.ch`;
    const blob = new Blob([generatedCode], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  if (!isOpen) {
    return (
      <button
        onClick={() => setIsOpen(true)}
        disabled={!diagramData}
        className={`px-4 py-2 rounded text-white ${
          !diagramData 
            ? 'bg-gray-400 cursor-not-allowed' 
            : 'bg-green-600 hover:bg-green-700'
        }`}
      >
        🔧 Generate .ch Code
      </button>
    );
  }

  return (
    <>
      {/* Modal Backdrop */}
      <div className="fixed inset-0 bg-black bg-opacity-50 z-50 flex items-center justify-center p-4">
        <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b">
            <h2 className="text-xl font-semibold text-gray-900">
              🔧 Chariot Code Generator
            </h2>
            <button
              onClick={() => setIsOpen(false)}
              className="text-gray-400 hover:text-gray-600 text-2xl"
            >
              ×
            </button>
          </div>

          {/* Content */}
          <div className="p-6">
            {/* Controls */}
            <div className="flex gap-3 mb-4">
              <button
                onClick={handleGenerate}
                disabled={isGenerating}
                className={`px-4 py-2 rounded text-white ${
                  isGenerating 
                    ? 'bg-gray-400 cursor-not-allowed' 
                    : 'bg-blue-600 hover:bg-blue-700'
                }`}
              >
                {isGenerating ? '🔄 Generating...' : '🚀 Generate Code'}
              </button>
              
              {generatedCode && (
                <>
                  <Button 
                    onClick={handleCopy}
                    className="bg-gray-600 hover:bg-gray-700 text-white"
                  >
                    📋 Copy
                  </Button>
                  <Button 
                    onClick={handleDownload}
                    className="bg-green-600 hover:bg-green-700 text-white"
                  >
                    💾 Download .ch
                  </Button>
                </>
              )}
            </div>

            {/* Error Display */}
            {error && (
              <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
                <strong>Error:</strong> {error}
              </div>
            )}

            {/* Code Display */}
            <div className="relative">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Generated Chariot Code:
              </label>
              <pre className="bg-gray-900 text-green-400 p-4 rounded-lg overflow-auto max-h-96 text-sm font-mono">
                {generatedCode || 'Click "Generate Code" to see the output...'}
              </pre>
            </div>

            {/* Stats */}
            {generatedCode && (
              <div className="mt-4 text-sm text-gray-600">
                <strong>Stats:</strong> {generatedCode.split('\n').length} lines, {' '}
                {(generatedCode.match(/\w+\(/g) || []).length} function calls
              </div>
            )}
          </div>

          {/* Footer */}
          <div className="flex justify-end gap-3 p-6 border-t bg-gray-50">
            <Button 
              onClick={() => setIsOpen(false)}
              className="bg-gray-600 hover:bg-gray-700 text-white"
            >
              Close
            </Button>
          </div>
        </div>
      </div>
    </>
  );
};

export default ChariotCodeGeneratorComponent;
