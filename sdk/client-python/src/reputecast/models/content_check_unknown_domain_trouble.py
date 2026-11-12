from enum import StrEnum

class ContentCheckUnknownDomainTrouble(StrEnum):
    OTHER = "other"
    REFUSED = "refused"
    TIMEOUT = "timeout"

    def __str__(self) -> str:
        return str(self.value)
