from enum import StrEnum

class ReferralCheckKey(StrEnum):
    EMAILS = "emails"
    FILES = "files"
    MILESTONES = "milestones"
    SENDER = "sender"
    STRIPE = "stripe"

    def __str__(self) -> str:
        return str(self.value)
