from enum import StrEnum

class WhatsAppStepEstimateInputMode(StrEnum):
    TEMPLATE = "template"
    TEXT = "text"
    VALUE_2 = ""

    def __str__(self) -> str:
        return str(self.value)
