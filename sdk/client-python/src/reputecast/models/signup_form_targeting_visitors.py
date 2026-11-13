from enum import StrEnum

class SignupFormTargetingVisitors(StrEnum):
    ALL = "all"
    IN_SEGMENT = "in_segment"
    NEW = "new"
    NOT_IN_SEGMENT = "not_in_segment"
    RETURNING = "returning"

    def __str__(self) -> str:
        return str(self.value)
