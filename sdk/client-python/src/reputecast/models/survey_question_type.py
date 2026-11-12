from enum import StrEnum

class SurveyQuestionType(StrEnum):
    EMOJI = "emoji"
    LONG = "long"
    MULTI = "multi"
    SCALE = "scale"
    SHORT = "short"
    SINGLE = "single"

    def __str__(self) -> str:
        return str(self.value)
