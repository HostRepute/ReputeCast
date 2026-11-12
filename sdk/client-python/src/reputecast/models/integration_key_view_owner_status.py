from enum import StrEnum

class IntegrationKeyViewOwnerStatus(StrEnum):
    ACTIVE = "active"
    DEACTIVATED = "deactivated"
    REMOVED = "removed"

    def __str__(self) -> str:
        return str(self.value)
