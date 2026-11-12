from enum import StrEnum

class ContactBulkFailureCode(StrEnum):
    COOLDOWN = "cooldown"
    REFUSED = "refused"
    SUPPRESSED = "suppressed"

    def __str__(self) -> str:
        return str(self.value)
