from enum import StrEnum

class EarnPriceOrigin(StrEnum):
    MAPPED = "mapped"
    TIER = "tier"

    def __str__(self) -> str:
        return str(self.value)
