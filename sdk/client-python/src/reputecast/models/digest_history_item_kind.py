from enum import StrEnum

class DigestHistoryItemKind(StrEnum):
    CREATED = "created"
    GOES_IN = "goes_in"
    LEFT_OUT = "left_out"
    NOT_SENT = "not_sent"
    PAUSED = "paused"
    SENT = "sent"
    SKIPPED = "skipped"
    TURNED_ON = "turned_on"

    def __str__(self) -> str:
        return str(self.value)
