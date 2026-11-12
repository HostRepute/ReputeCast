from enum import StrEnum

class ListContactsCheck(StrEnum):
    INVALID = "invalid"
    NEVER = "never"
    RISKY = "risky"
    UNKNOWN = "unknown"
    VALID = "valid"
    VALUE_5 = ""

    def __str__(self) -> str:
        return str(self.value)
