import React from "react";

type InputProps = {
  type?: string;
  value?: string;
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  placeholder?: string;
  className?: string;
  maxLength?: number;
};

export const Input = ({ type = "text", value, onChange, placeholder, className = "", maxLength }: InputProps) => (
  <input
    type={type}
    value={value}
    onChange={onChange}
    placeholder={placeholder}
    maxLength={maxLength}
    className={`px-3 py-1 border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500 ${className}`}
  />
);
