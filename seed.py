"""
Seed script for PQAP — creates initial admin user and default data.

Usage:
    python seed.py
    python seed.py --username admin --password YourSecurePass123
    python seed.py --reset   # drops and recreates tables
    python seed.py --force   # reset password non-interactively if user exists

Requires: POSTGRES_URL env var (or defaults to localhost)
"""

import argparse
import asyncio
import os
import sys

import asyncpg
from passlib.context import CryptContext

POSTGRES_URL = os.getenv("POSTGRES_URL", "postgres://localhost:5432/pqap")
DEFAULT_USERNAME = "admin"
DEFAULT_PASSWORD = "PQAP@dm1n2026!"

pwd_context = CryptContext(schemes=["bcrypt"], deprecated="auto", bcrypt__rounds=12)

USERS_TABLE = """
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
"""

ACCOUNTS_TABLE = """
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    wallet_address VARCHAR(100) NOT NULL,
    private_key_encrypted BYTEA NOT NULL,
    private_key_iv BYTEA NOT NULL,
    private_key_tag BYTEA NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_accounts_wallet ON accounts(wallet_address);
CREATE INDEX IF NOT EXISTS idx_accounts_active ON accounts(is_active);
"""

RISK_STATE_INIT = """
INSERT INTO risk_parameters (daily_loss_limit, max_position_per_market, max_position_per_strategy, updated_at)
VALUES (2.0, 10.0, 20.0, NOW())
ON CONFLICT DO NOTHING;
"""

RISK_PARAMETERS_TABLE = """
CREATE TABLE IF NOT EXISTS risk_parameters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    daily_loss_limit NUMERIC(12,4),
    max_position_per_market NUMERIC(12,4),
    max_position_per_strategy NUMERIC(12,4),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_risk_parameters_updated_at ON risk_parameters (updated_at DESC);
"""

ACCOUNT_RISK_LIMITS_TABLE = """
CREATE TABLE IF NOT EXISTS account_risk_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    daily_loss_limit DECIMAL(10,4) NOT NULL DEFAULT 2.0,
    max_position_per_market DECIMAL(10,4) NOT NULL DEFAULT 10.0,
    max_position_per_strategy DECIMAL(10,4) NOT NULL DEFAULT 20.0,
    drawdown_threshold DECIMAL(10,4) NOT NULL DEFAULT 10.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(account_id)
);
"""


async def seed(username: str, password: str, reset: bool = False, force: bool = False):
    print(f"Connecting to: {POSTGRES_URL.split('@')[-1] if '@' in POSTGRES_URL else POSTGRES_URL}")

    try:
        conn = await asyncpg.connect(POSTGRES_URL)
    except Exception as e:
        print(f"ERROR: Cannot connect to PostgreSQL — {e}")
        print("\nMake sure PostgreSQL is running and POSTGRES_URL is set correctly.")
        print(f"  Current: {POSTGRES_URL}")
        sys.exit(1)

    try:
        if reset:
            print("Dropping existing tables...")
            await conn.execute("DROP TABLE IF EXISTS account_risk_limits CASCADE")
            await conn.execute("DROP TABLE IF EXISTS risk_parameters CASCADE")
            await conn.execute("DROP TABLE IF EXISTS accounts CASCADE")
            await conn.execute("DROP TABLE IF EXISTS users CASCADE")

        # Create tables
        print("Creating tables...")
        await conn.execute(USERS_TABLE)
        await conn.execute(ACCOUNTS_TABLE)
        await conn.execute(RISK_PARAMETERS_TABLE)
        await conn.execute(ACCOUNT_RISK_LIMITS_TABLE)

        # Check if admin already exists
        existing = await conn.fetchrow(
            "SELECT id, username, role FROM users WHERE username = $1",
            username,
        )

        if existing:
            print(f"User '{username}' already exists (id={existing['id']}, role={existing['role']})")
            if force:
                password_hash = pwd_context.hash(password)
                await conn.execute(
                    "UPDATE users SET password_hash = $1 WHERE username = $2",
                    password_hash,
                    username,
                )
                print(f"Password updated for '{username}'")
            else:
                print("Use --force to reset password non-interactively.")
        else:
            # Create admin user
            password_hash = pwd_context.hash(password)
            row = await conn.fetchrow(
                "INSERT INTO users (username, password_hash, role) VALUES ($1, $2, 'admin') RETURNING id, username, role",
                username,
                password_hash,
            )
            print(f"Admin user created:")
            print(f"  ID:       {row['id']}")
            print(f"  Username: {row['username']}")
            print(f"  Role:     {row['role']}")

        # Initialize risk parameters if empty
        count = await conn.fetchval("SELECT COUNT(*) FROM risk_parameters")
        if count == 0:
            await conn.execute(RISK_STATE_INIT)
            print("Default risk parameters initialized")

        print("\nSeed completed successfully!")

    finally:
        await conn.close()


def main():
    parser = argparse.ArgumentParser(description="Seed PQAP database")
    parser.add_argument("--username", default=DEFAULT_USERNAME, help="Admin username")
    parser.add_argument("--password", default=DEFAULT_PASSWORD, help="Admin password")
    parser.add_argument("--reset", action="store_true", help="Drop and recreate tables")
    parser.add_argument("--force", action="store_true", help="Reset password non-interactively if user exists")
    args = parser.parse_args()

    if len(args.password) < 12:
        print("ERROR: Password must be at least 12 characters")
        sys.exit(1)

    asyncio.run(seed(args.username, args.password, args.reset, args.force))


if __name__ == "__main__":
    main()
