from enum import StrEnum

class SurveyCloseMode(StrEnum):
    DATE = "date"
    MANUAL = "manual"

    def __str__(self) -> str:
        return str(self.value)
