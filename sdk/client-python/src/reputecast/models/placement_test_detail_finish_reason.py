from enum import StrEnum

class PlacementTestDetailFinishReason(StrEnum):
    ALL_FOUND = "all_found"
    STOPPED = "stopped"
    TIMEOUT = "timeout"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
