from enum import StrEnum

class AICapViewKind(StrEnum):
    NONE = "none"
    TOKENS = "tokens"
    USD = "usd"

    def __str__(self) -> str:
        return str(self.value)
