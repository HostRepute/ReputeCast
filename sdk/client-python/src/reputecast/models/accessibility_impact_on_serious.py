from enum import StrEnum

class AccessibilityImpactOnSerious(StrEnum):
    BLOCK = "block"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
