from enum import StrEnum

class WhatsAppEventKind(StrEnum):
    ACCOUNT = "account"
    MESSAGE = "message"
    QUALITY = "quality"
    STATUS = "status"
    TEMPLATE = "template"
    UNKNOWN = "unknown"

    def __str__(self) -> str:
        return str(self.value)
