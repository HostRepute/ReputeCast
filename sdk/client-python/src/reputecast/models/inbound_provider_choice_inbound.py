from enum import StrEnum

class InboundProviderChoiceInbound(StrEnum):
    AUTOMATIC = "automatic"
    MANUAL = "manual"
    VALUE_0 = ""

    def __str__(self) -> str:
        return str(self.value)
