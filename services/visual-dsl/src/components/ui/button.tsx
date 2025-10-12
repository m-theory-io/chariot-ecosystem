import React from "react";

type ButtonProps = {
  children: React.ReactNode;
  onClick?: (e?: React.MouseEvent<HTMLButtonElement>) => void;
  onDoubleClick?: () => void;
  className?: string;
  title?: string;
  disabled?: boolean;
};

export const Button = ({ children, onClick, onDoubleClick, className = "", title, disabled }: ButtonProps) => (
  <button
    onClick={onClick}
    onDoubleClick={onDoubleClick}
    disabled={disabled}
    className={`px-4 py-2 rounded bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 ${disabled ? 'opacity-60 cursor-not-allowed' : ''} ${className}`}
    title={title}
  >
    {children}
  </button>
);
