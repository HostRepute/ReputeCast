from enum import StrEnum

class CommerceSyncStateRevokedReason(StrEnum):
    REVOKED = "revoked"
    UNINSTALLED = "uninstalled"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
