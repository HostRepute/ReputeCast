from enum import StrEnum

class IntegrationAIFactsCapKind(StrEnum):
    NONE = "none"
    TOKENS = "tokens"
    USD = "usd"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
