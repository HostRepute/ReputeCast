from enum import StrEnum

class WhatsAppTemplateCategory(StrEnum):
    AUTHENTICATION = "authentication"
    MARKETING = "marketing"
    UTILITY = "utility"

    def __str__(self) -> str:
        return str(self.value)
