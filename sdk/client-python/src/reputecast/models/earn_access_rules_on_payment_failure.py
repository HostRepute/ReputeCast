from enum import StrEnum

class EarnAccessRulesOnPaymentFailure(StrEnum):
    END = "end"
    KEEP = "keep"

    def __str__(self) -> str:
        return str(self.value)
