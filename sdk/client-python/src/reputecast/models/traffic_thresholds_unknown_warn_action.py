from enum import StrEnum

class TrafficThresholdsUnknownWarnAction(StrEnum):
    PAUSE_SEND = "pause_send"
    SLOW = "slow"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
