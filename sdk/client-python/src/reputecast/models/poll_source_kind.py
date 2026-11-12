from enum import StrEnum

class PollSourceKind(StrEnum):
    POLL = "poll"
    SURVEY = "survey"

    def __str__(self) -> str:
        return str(self.value)
