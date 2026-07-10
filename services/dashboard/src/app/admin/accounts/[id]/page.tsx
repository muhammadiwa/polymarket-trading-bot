"use client";

import Link from "next/link";
import { useRouter, useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import type { Account, AccountUpdateRequest } from "@/types";
import { fetchAccount, updateAccount, deactivateAccount, activateAccount } from "@/lib/api";

export default function EditAccountPage() {
  const router = useRouter();
  const params = useParams();
  const accountId = params.id as string;

  const [account, setAccount] = useState<Account | null>(null);
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const loadAccount = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await fetchAccount(accountId);
      setAccount(data);
      setName(data.name);
    } catch (err) {
      setError("Failed to load account");
    } finally {
      setLoading(false);
    }
  }, [accountId]);

  useEffect(() => {
    loadAccount();
  }, [loadAccount]);

  const handleSave = async () => {
    if (!name.trim()) {
      setError("Name is required");
      return;
    }

    try {
      setSaving(true);
      setError(null);
      setSuccess(null);

      const request: AccountUpdateRequest = { name: name.trim() };
      await updateAccount(accountId, request);
      setSuccess("Account updated successfully");
      loadAccount();
    } catch (err: any) {
      setError(err.message || "Failed to update account");
    } finally {
      setSaving(false);
    }
  };

  const handleToggleActive = async () => {
    if (!account) return;

    try {
      setError(null);
      setSuccess(null);

      if (account.isActive) {
        await deactivateAccount(accountId);
        setSuccess("Account deactivated");
      } else {
        await activateAccount(accountId);
        setSuccess("Account activated");
      }

      loadAccount();
    } catch (err: any) {
      setError(err.message || "Failed to update account status");
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-gray-400">Loading account...</div>
      </div>
    );
  }

  if (!account) {
    return (
      <div className="space-y-4">
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">
          Account not found
        </div>
        <Link href="/admin/accounts" className="text-blue-400 hover:underline">
          ← Back to Accounts
        </Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/admin/accounts" className="text-gray-400 hover:text-white">
          ← Accounts
        </Link>
        <h2 className="text-xl font-semibold text-white">Edit Account</h2>
      </div>

      {error && (
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">{error}</div>
      )}
      {success && (
        <div className="rounded-md bg-green-900/50 p-4 text-green-200">{success}</div>
      )}

      <div className="max-w-lg space-y-6">
        {/* Account Info */}
        <div className="rounded-lg border border-gray-800 bg-gray-900 p-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <span className="text-sm text-gray-400">Status</span>
              <p className={`font-medium ${account.isActive ? "text-green-400" : "text-gray-400"}`}>
                {account.isActive ? "Active" : "Inactive"}
              </p>
            </div>
            <div>
              <span className="text-sm text-gray-400">Created</span>
              <p className="text-white">{new Date(account.createdAt).toLocaleString()}</p>
            </div>
            <div className="col-span-2">
              <span className="text-sm text-gray-400">Wallet Address</span>
              <p className="font-mono text-white">{account.walletAddress}</p>
            </div>
          </div>
        </div>

        {/* Edit Form */}
        <div className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400">Account Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="mt-1 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
            />
          </div>

          <div className="flex gap-4">
            <button
              onClick={handleSave}
              disabled={saving}
              className="rounded-md bg-blue-600 px-6 py-2 text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {saving ? "Saving..." : "Save Changes"}
            </button>
            <button
              onClick={handleToggleActive}
              className={`rounded-md px-6 py-2 ${
                account.isActive
                  ? "bg-red-600 text-white hover:bg-red-700"
                  : "bg-green-600 text-white hover:bg-green-700"
              }`}
            >
              {account.isActive ? "Deactivate" : "Activate"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
