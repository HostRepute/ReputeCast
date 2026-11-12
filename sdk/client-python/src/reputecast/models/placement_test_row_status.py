from enum import StrEnum

class PlacementTestRowStatus(StrEnum):
    FINISHED = "finished"
    SENDING = "sending"
    STOPPED = "stopped"
    WAITING = "waiting"

    def __str__(self) -> str:
        return str(self.value)
