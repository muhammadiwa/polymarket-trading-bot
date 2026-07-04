"use client";

import { type ReactNode } from "react";

export interface Column<T> {
  key: string;
  header: string;
  sortable?: boolean;
  render: (row: T) => ReactNode;
  sortValue?: (row: T) => number | string;
}

interface TableProps<T> {
  columns: Column<T>[];
  data: T[];
  sortKey?: string;
  sortDir?: "asc" | "desc";
  onSort?: (key: string) => void;
  emptyMessage?: string;
}

export function Table<T extends { id: string }>({
  columns,
  data,
  sortKey,
  sortDir,
  onSort,
  emptyMessage = "No data",
}: TableProps<T>) {
  if (data.length === 0) {
    return (
      <div className="rounded-xl border border-white/10 bg-white/5 backdrop-blur-md p-8 text-center text-gray-500">
        {emptyMessage}
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-xl border border-white/10 bg-white/5 backdrop-blur-md">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-white/10">
            {columns.map((col) => (
              <th
                key={col.key}
                className={`px-4 py-3 text-left text-xs font-medium text-gray-400 uppercase tracking-wider ${
                  col.sortable ? "cursor-pointer select-none hover:text-gray-200" : ""
                }`}
                onClick={col.sortable ? () => onSort?.(col.key) : undefined}
              >
                <span className="inline-flex items-center gap-1">
                  {col.header}
                  {sortKey === col.key && (
                    <span>{sortDir === "asc" ? "▲" : "▼"}</span>
                  )}
                </span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-white/5">
          {data.map((row) => (
            <tr key={row.id} className="hover:bg-white/5 transition-colors">
              {columns.map((col) => (
                <td key={col.key} className="px-4 py-3 font-mono">
                  {col.render(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
