from enum import StrEnum

class ReferralSignalKind(StrEnum):
    BURST = "burst"
    NUMBERED = "numbered"
    SHARED_IP = "shared_ip"

    def __str__(self) -> str:
        return str(self.value)
