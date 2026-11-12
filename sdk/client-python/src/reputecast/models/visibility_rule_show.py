from enum import StrEnum

class VisibilityRuleShow(StrEnum):
    EVERYONE = "everyone"
    EXCEPT = "except"
    NOBODY = "nobody"
    ONLY = "only"

    def __str__(self) -> str:
        return str(self.value)
