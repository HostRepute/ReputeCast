from enum import StrEnum

class SuppressCampaignReportBouncesType(StrEnum):
    BLOCKED = "blocked"
    DEFERRAL_EXPIRED = "deferral_expired"
    DOMAIN_INVALID = "domain_invalid"
    MAILBOX_FULL = "mailbox_full"
    OTHER = "other"
    UNKNOWN_USER = "unknown_user"

    def __str__(self) -> str:
        return str(self.value)
