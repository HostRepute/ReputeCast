from enum import StrEnum

class WhatsAppThreadItemKind(StrEnum):
    BUTTON = "button"
    KEYWORD = "keyword"
    REPLY = "reply"
    TEMPLATE = "template"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
