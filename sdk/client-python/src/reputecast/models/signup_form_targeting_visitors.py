from enum import StrEnum

class SignupFormTargetingVisitors(StrEnum):
    ALL = "all"
    NEW = "new"
    RETURNING = "returning"

    def __str__(self) -> str:
        return str(self.value)
