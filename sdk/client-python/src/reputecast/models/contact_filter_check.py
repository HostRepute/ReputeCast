from enum import StrEnum

class ContactFilterCheck(StrEnum):
    INVALID = "invalid"
    NEVER = "never"
    RISKY = "risky"
    UNKNOWN = "unknown"
    VALID = "valid"

    def __str__(self) -> str:
        return str(self.value)
