from enum import StrEnum

class FieldSourceKind(StrEnum):
    POLL = "poll"
    SURVEY = "survey"

    def __str__(self) -> str:
        return str(self.value)
