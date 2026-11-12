from enum import StrEnum

class IntegrationManagedKeyStatus(StrEnum):
    ACTIVE = "active"
    EXPIRED = "expired"
    GRACE = "grace"
    REVOKED = "revoked"

    def __str__(self) -> str:
        return str(self.value)
