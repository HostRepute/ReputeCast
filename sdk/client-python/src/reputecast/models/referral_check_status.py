from enum import StrEnum

class ReferralCheckStatus(StrEnum):
    FAIL = "fail"
    PASS = "pass"

    def __str__(self) -> str:
        return str(self.value)
