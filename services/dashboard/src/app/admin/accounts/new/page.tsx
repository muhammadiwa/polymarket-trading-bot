"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import type { AccountCreateRequest } from "@/types";
import { createAccount } from "@/lib/api";

export default function NewAccountPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [walletAddress, setWalletAddress] = useState("");
  const [privateKey, setPrivateKey] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name || !walletAddress || !privateKey) {
      setError("All fields are required");
      return;
    }

    if (!walletAddress.match(/^0x[0-9a-fA-F]{40}$/)) {
      setError("Invalid wallet address format (must be 0x followed by 40 hex characters)");
      return;
    }

    try {
      setSaving(true);
      setError(null);

      const request: AccountCreateRequest = {
        name,
        walletAddress,
        privateKey,
      };

      await createAccount(request);
      router.push("/admin/accounts");
    } catch (err: any) {
      setError(err.message || "Failed to create account");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/admin/accounts" className="text-gray-400 hover:text-white">
          ← Accounts
        </Link>
        <h2 className="text-xl font-semibold text-white">New Account</h2>
      </div>

      {error && (
        <div className="rounded-md bg-red-900/50 p-4 text-red-200">{error}</div>
      )}

      <form onSubmit={handleSubmit} className="max-w-lg space-y-4">
        <div>
          <label className="block text-sm text-gray-400">Account Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., Main Trading Account"
            className="mt-1 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 text-white"
            required
          />
        </div>

        <div>
          <label className="block text-sm text-gray-400">Wallet Address</label>
          <input
            type="text"
            value={walletAddress}
            onChange={(e) => setWalletAddress(e.target.value)}
            placeholder="0x..."
            className="mt-1 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 font-mono text-white"
            required
          />
          <p className="mt-1 text-xs text-gray-500">
            Ethereum wallet address (0x + 40 hex characters)
          </p>
        </div>

        <div>
          <label className="block text-sm text-gray-400">Private Key</label>
          <input
            type="password"
            value={privateKey}
            onChange={(e) => setPrivateKey(e.target.value)}
            placeholder="Enter private key"
            className="mt-1 w-full rounded-md border border-gray-700 bg-gray-800 px-3 py-2 font-mono text-white"
            required
          />
          <p className="mt-1 text-xs text-gray-500">
            Private key will be encrypted and stored securely
          </p>
        </div>

        <div className="flex gap-4">
          <button
            type="submit"
            disabled={saving}
            className="rounded-md bg-blue-600 px-6 py-2 text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {saving ? "Creating..." : "Create Account"}
          </button>
          <Link
            href="/admin/accounts"
            className="rounded-md bg-gray-800 px-6 py-2 text-gray-300 hover:bg-gray-700"
          >
            Cancel
          </Link>
        </div>
      </form>
    </div>
  );
}
