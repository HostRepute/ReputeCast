from enum import StrEnum

class EarnPriceStatus(StrEnum):
    CURRENT = "current"
    LEGACY = "legacy"

    def __str__(self) -> str:
        return str(self.value)
