from enum import StrEnum

class DigestFeedStatusKind(StrEnum):
    ATOM = "atom"
    JSON = "json"
    RSS = "rss"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
