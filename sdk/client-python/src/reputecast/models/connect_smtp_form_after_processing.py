from enum import StrEnum

class ConnectSMTPFormAfterProcessing(StrEnum):
    DELETE = "delete"
    LEAVE = "leave"
    MOVE = "move"

    def __str__(self) -> str:
        return str(self.value)
