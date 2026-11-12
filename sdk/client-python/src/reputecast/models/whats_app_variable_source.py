from enum import StrEnum

class WhatsAppVariableSource(StrEnum):
    FIELD = "field"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
