from enum import StrEnum

class SurveyLinkVia(StrEnum):
    LINK = "link"
    POLL = "poll"

    def __str__(self) -> str:
        return str(self.value)
