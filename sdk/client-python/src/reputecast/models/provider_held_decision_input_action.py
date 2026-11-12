from enum import StrEnum

class ProviderHeldDecisionInputAction(StrEnum):
    CANCEL = "cancel"
    MOVE = "move"
    WAIT = "wait"

    def __str__(self) -> str:
        return str(self.value)
