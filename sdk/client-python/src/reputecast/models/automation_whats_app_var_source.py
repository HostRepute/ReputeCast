from enum import StrEnum

class AutomationWhatsAppVarSource(StrEnum):
    FIELD = "field"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
