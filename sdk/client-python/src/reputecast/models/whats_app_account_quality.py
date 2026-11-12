from enum import StrEnum

class WhatsAppAccountQuality(StrEnum):
    HIGH = "high"
    LOW = "low"
    MEDIUM = "medium"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)
