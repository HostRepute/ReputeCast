from enum import StrEnum

class ScoringRuleOp(StrEnum):
    IS_NOT = "is_not"
    IS_ONE_OF = "is_one_of"
    IS_SET = "is_set"

    def __str__(self) -> str:
        return str(self.value)
