from enum import StrEnum

class EarnImportDecisionsNoSubscription(StrEnum):
    COMPLIMENTARY = "complimentary"
    FREE = "free"
    SKIP = "skip"

    def __str__(self) -> str:
        return str(self.value)
