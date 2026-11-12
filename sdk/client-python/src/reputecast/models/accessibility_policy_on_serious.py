from enum import StrEnum

class AccessibilityPolicyOnSerious(StrEnum):
    BLOCK = "block"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
