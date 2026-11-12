from enum import StrEnum

class AutomationTriggerAnswerOp(StrEnum):
    ANY = "any"
    IS = "is"
    IS_NOT = "is_not"

    def __str__(self) -> str:
        return str(self.value)
