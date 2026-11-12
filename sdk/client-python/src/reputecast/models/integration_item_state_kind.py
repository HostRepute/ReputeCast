from enum import StrEnum

class IntegrationItemStateKind(StrEnum):
    ACTION = "action"
    TRIGGER = "trigger"

    def __str__(self) -> str:
        return str(self.value)
