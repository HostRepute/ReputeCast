from enum import StrEnum

class EarnConnectionProvider(StrEnum):
    STRIPE = "stripe"

    def __str__(self) -> str:
        return str(self.value)
