from enum import StrEnum

class TrafficThresholdsBlockAction(StrEnum):
    PAUSE_MAILBOX = "pause_mailbox"
    PAUSE_SEND = "pause_send"
    SLOW = "slow"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
