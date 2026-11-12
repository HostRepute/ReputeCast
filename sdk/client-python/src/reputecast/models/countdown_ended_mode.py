from enum import StrEnum

class CountdownEndedMode(StrEnum):
    TEXT = "text"
    ZEROS = "zeros"

    def __str__(self) -> str:
        return str(self.value)
