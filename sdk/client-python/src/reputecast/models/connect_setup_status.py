from enum import StrEnum

class ConnectSetupStatus(StrEnum):
    ABANDONED = "abandoned"
    FINISHED = "finished"
    OPEN = "open"

    def __str__(self) -> str:
        return str(self.value)
