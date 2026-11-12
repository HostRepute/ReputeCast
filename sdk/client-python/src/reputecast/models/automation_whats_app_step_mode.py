from enum import StrEnum

class AutomationWhatsAppStepMode(StrEnum):
    TEMPLATE = "template"
    TEXT = "text"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
