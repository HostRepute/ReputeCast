from enum import StrEnum

class PollSourcePollKind(StrEnum):
    CHOICE = "choice"
    EMOJI = "emoji"
    SCALE = "scale"

    def __str__(self) -> str:
        return str(self.value)
