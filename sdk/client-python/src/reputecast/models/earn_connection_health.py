from enum import StrEnum

class EarnConnectionHealth(StrEnum):
    CONNECTED = "connected"
    KEY_REJECTED = "key_rejected"
    MISSING_PERMISSIONS = "missing_permissions"
    SETUP = "setup"
    WEBHOOK_MISSING = "webhook_missing"
    WEBHOOK_SILENT = "webhook_silent"

    def __str__(self) -> str:
        return str(self.value)
