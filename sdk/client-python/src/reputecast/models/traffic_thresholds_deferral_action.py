from enum import StrEnum

class TrafficThresholdsDeferralAction(StrEnum):
    PAUSE_MAILBOX = "pause_mailbox"
    SLOW = "slow"
    WARN = "warn"

    def __str__(self) -> str:
        return str(self.value)
