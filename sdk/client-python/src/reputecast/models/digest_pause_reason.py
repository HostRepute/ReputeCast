from enum import StrEnum

class DigestPauseReason(StrEnum):
    FEED = "feed"
    OWNER = "owner"
    PERSON = "person"

    def __str__(self) -> str:
        return str(self.value)
