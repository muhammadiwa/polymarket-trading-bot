import { type ReactNode } from "react";

interface CardProps {
  title: string;
  children: ReactNode;
  className?: string;
}

export function Card({ title, children, className = "" }: CardProps) {
  return (
    <div
      className={`rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-5 ${className}`}
    >
      <h3 className="text-sm font-medium text-gray-400 mb-2">{title}</h3>
      {children}
    </div>
  );
}
