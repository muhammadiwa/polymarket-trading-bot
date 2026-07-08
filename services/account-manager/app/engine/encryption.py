import hashlib
import logging
import os

from cryptography.exceptions import InvalidTag
from cryptography.hazmat.primitives.ciphers.aead import AESGCM

logger = logging.getLogger(__name__)


class WalletEncryption:
    def __init__(self, master_key: str):
        self.master_key = master_key.encode() if isinstance(master_key, str) else master_key

    def encrypt_private_key(self, private_key: str, account_id: str) -> dict:
        """Encrypt private key using AES-256-GCM."""
        if not private_key:
            raise ValueError("private_key cannot be empty")
        if not account_id:
            raise ValueError("account_id cannot be empty")

        key = self._derive_key(account_id)
        iv = os.urandom(12)

        aesgcm = AESGCM(key)
        ciphertext = aesgcm.encrypt(iv, private_key.encode(), None)

        # AESGCM.encrypt returns ciphertext + 16-byte tag appended
        encrypted = ciphertext[:-16]
        tag = ciphertext[-16:]

        return {
            "encrypted": encrypted,
            "iv": iv,
            "tag": tag,
        }

    def decrypt_private_key(self, encrypted: bytes, iv: bytes, tag: bytes, account_id: str) -> str:
        """Decrypt private key."""
        if not encrypted or not iv or not tag:
            raise ValueError("encrypted, iv, and tag cannot be empty")
        if not account_id:
            raise ValueError("account_id cannot be empty")
        if len(iv) != 12:
            raise ValueError("iv must be exactly 12 bytes")

        key = self._derive_key(account_id)

        try:
            aesgcm = AESGCM(key)
            ciphertext = encrypted + tag
            plaintext = aesgcm.decrypt(iv, ciphertext, None)
            return plaintext.decode()
        except InvalidTag:
            logger.warning("decryption failed: invalid tag", extra={"account_id": account_id})
            raise ValueError("Decryption failed: invalid key or corrupted data")

    def _derive_key(self, account_id: str) -> bytes:
        """Derive encryption key from master key + account ID."""
        return hashlib.sha256(self.master_key + account_id.encode()).digest()
