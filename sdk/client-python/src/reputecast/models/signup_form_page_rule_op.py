from enum import StrEnum

class SignupFormPageRuleOp(StrEnum):
    CONTAINS = "contains"
    EQUALS = "equals"
    STARTS_WITH = "starts_with"

    def __str__(self) -> str:
        return str(self.value)
