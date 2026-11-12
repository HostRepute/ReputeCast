from enum import StrEnum

class APIKeyViewStatus(StrEnum):
    ACTIVE = "active"
    EXPIRED = "expired"

    def __str__(self) -> str:
        return str(self.value)
