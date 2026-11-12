from enum import StrEnum

class GetReferralReviewSignal(StrEnum):
    BURST = "burst"
    NUMBERED = "numbered"
    SHARED_IP = "shared_ip"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
