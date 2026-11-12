from enum import StrEnum

class SignupFormPageRulesMode(StrEnum):
    ALL = "all"
    EXCEPT = "except"
    ONLY = "only"

    def __str__(self) -> str:
        return str(self.value)
