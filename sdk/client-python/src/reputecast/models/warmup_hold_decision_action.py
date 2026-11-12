from enum import StrEnum

class WarmupHoldDecisionAction(StrEnum):
    CANCEL = "cancel"
    MOVE = "move"
    WAIT = "wait"

    def __str__(self) -> str:
        return str(self.value)
