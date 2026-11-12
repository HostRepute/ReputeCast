from enum import StrEnum

class ConnectFieldType(StrEnum):
    NUMBER = "number"
    TEXT = "text"

    def __str__(self) -> str:
        return str(self.value)
