from enum import StrEnum

class ContactAddressCheckResult(StrEnum):
    INVALID = "invalid"
    RISKY = "risky"
    UNKNOWN = "unknown"
    VALID = "valid"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
