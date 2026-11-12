from enum import StrEnum

class TrafficThresholdsComplaintAction(StrEnum):
    PAUSE_MAILBOX = "pause_mailbox"
    PAUSE_SEND = "pause_send"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
