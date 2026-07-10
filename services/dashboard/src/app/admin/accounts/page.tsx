"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import type { Account } from "@/types";
import { fetchAccounts, deactivateAccount, activateAccount } from "@/lib/api";

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const loadAccounts = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchAccounts();
      setAccounts(data.accounts);
      setTotal(data.total);
    } catch (err) {
      setError("Failed to load accounts");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadAccounts();
  }, [loadAccounts]);

  const handleToggleActive = async (account: Account) => {
    try {
      setError(null);
      setSuccess(null);

      if (account.isActive) {
        await deactivateAccount(account.id);
        setSuccess(`Account "${account.name}" deactivated`);
      } else {
        await activateAccount(account.id);
        setSuccess(`Account "${account.name}" activated`);
      }

      loadAccounts();
    } catch (err: any) {
      setError(err.message || "Failed to update account");
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/admin" className="text-gray-400 hover:text-white">
            ← Admin
          </Link>
          <h2 className="text-xl font-semibold text-white">Account Management</h2>
        </div>
        <Link
          href="/admin/accounts/new"
          className="rounded-md bg-blue-600 px-4 py-2 text-sm text-white hover:bg-blue-700"
        >
          + New Account
        </Link>
      </div>

      {error && (
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">{error}</div>
      )}
      {success && (
        <div className="rounded-md bg-green-900/50 p-4 text-green-200">{success}</div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-2 gap-4 md:grid-cols-3">
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
          <span className="text-sm text-gray-400">Total Accounts</span>
          <p className="text-2xl font-bold text-white">{total}</p>
        </div>
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
          <span className="text-sm text-gray-400">Active</span>
          <p className="text-2xl font-bold text-green-400">
            {accounts.filter((a) => a.isActive).length}
          </p>
        </div>
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
          <span className="text-sm text-gray-400">Inactive</span>
          <p className="text-2xl font-bold text-gray-400">
            {accounts.filter((a) => !a.isActive).length}
          </p>
        </div>
      </div>

      {/* Accounts Table */}
      <div className="rounded-lg border border-gray-800 bg-gray-900">
        {loading ? (
          <div className="p-8 text-center text-gray-400">Loading...</div>
        ) : accounts.length === 0 ? (
          <div className="p-8 text-center text-gray-400">
            No accounts found. Create your first account to get started.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-800 text-left text-gray-400">
                  <th className="p-4">Name</th>
                  <th className="p-4">Wallet Address</th>
                  <th className="p-4">Status</th>
                  <th className="p-4">Created</th>
                  <th className="p-4">Actions</th>
                </tr>
              </thead>
              <tbody>
                {accounts.map((account) => (
                  <tr
                    key={account.id}
                    className="border-b border-gray-800 hover:bg-gray-800/50"
                  >
                    <td className="p-4">
                      <Link
                        href={`/admin/accounts/${account.id}`}
                        className="text-white hover:text-blue-400"
                      >
                        {account.name}
                      </Link>
                    </td>
                    <td className="p-4">
                      <span className="font-mono text-gray-400">
                        {account.walletAddress.slice(0, 6)}...{account.walletAddress.slice(-4)}
                      </span>
                    </td>
                    <td className="p-4">
                      <span
                        className={`rounded px-2 py-0.5 text-xs ${
                          account.isActive
                            ? "bg-green-900 text-green-400"
                            : "bg-gray-800 text-gray-400"
                        }`}
                      >
                        {account.isActive ? "Active" : "Inactive"}
                      </span>
                    </td>
                    <td className="p-4 text-gray-400">
                      {new Date(account.createdAt).toLocaleDateString()}
                    </td>
                    <td className="p-4">
                      <div className="flex gap-2">
                        <Link
                          href={`/admin/accounts/${account.id}`}
                          className="rounded bg-gray-800 px-3 py-1 text-xs text-gray-300 hover:bg-gray-700"
                        >
                          Edit
                        </Link>
                        <button
                          onClick={() => handleToggleActive(account)}
                          className={`rounded px-3 py-1 text-xs ${
                            account.isActive
                              ? "bg-red-800 text-red-300 hover:bg-red-700"
                              : "bg-green-800 text-green-300 hover:bg-green-700"
                          }`}
                        >
                          {account.isActive ? "Deactivate" : "Activate"}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
