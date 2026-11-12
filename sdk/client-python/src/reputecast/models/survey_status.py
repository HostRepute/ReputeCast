from enum import StrEnum

class SurveyStatus(StrEnum):
    CLOSED = "closed"
    DRAFT = "draft"
    OPEN = "open"

    def __str__(self) -> str:
        return str(self.value)
