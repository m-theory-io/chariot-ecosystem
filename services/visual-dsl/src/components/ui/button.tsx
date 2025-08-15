import React from "react";

type ButtonProps = {
  children: React.ReactNode;
  onClick?: (e?: React.MouseEvent<HTMLButtonElement>) => void;
  onDoubleClick?: () => void;
  className?: string;
  title?: string;
};

export const Button = ({ children, onClick, onDoubleClick, className = "", title }: ButtonProps) => (
  <button
    onClick={onClick}
    onDoubleClick={onDoubleClick}
    className={`px-4 py-2 rounded bg-blue-600 text-white hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 ${className}`}
    title={title}
  >
    {children}
  </button>
);
