from enum import StrEnum

class PlacementTestDetailStatus(StrEnum):
    FINISHED = "finished"
    SENDING = "sending"
    STOPPED = "stopped"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
