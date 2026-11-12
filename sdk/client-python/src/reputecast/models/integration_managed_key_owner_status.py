from enum import StrEnum

class IntegrationManagedKeyOwnerStatus(StrEnum):
    ACTIVE = "active"
    DEACTIVATED = "deactivated"
    REMOVED = "removed"

    def __str__(self) -> str:
        return str(self.value)
