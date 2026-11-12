from enum import StrEnum

class ReviewCanaryReasonsItem(StrEnum):
    NEVER_MAILED = "never_mailed"
    NEW_DOMAIN = "new_domain"
    NEW_IP = "new_ip"
    OVER_USUAL = "over_usual"

    def __str__(self) -> str:
        return str(self.value)
