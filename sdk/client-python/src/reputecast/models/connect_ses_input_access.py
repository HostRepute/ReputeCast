from enum import StrEnum

class ConnectSESInputAccess(StrEnum):
    KEYS = "keys"
    ROLE = "role"

    def __str__(self) -> str:
        return str(self.value)
