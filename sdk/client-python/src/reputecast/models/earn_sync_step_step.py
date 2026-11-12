from enum import StrEnum

class EarnSyncStepStep(StrEnum):
    MONTH = "month"
    PRODUCT = "product"
    YEAR = "year"

    def __str__(self) -> str:
        return str(self.value)
