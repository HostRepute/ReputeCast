from enum import StrEnum

class WhatsAppScheduleMode(StrEnum):
    AT = "at"
    LOCAL = "local"
    NOW = "now"

    def __str__(self) -> str:
        return str(self.value)
