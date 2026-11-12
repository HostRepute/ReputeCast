from enum import StrEnum

class IntegrationReasonCode(StrEnum):
    CALLS_FAILING = "calls_failing"
    DELIVERIES_FAILING = "deliveries_failing"
    KEY_EXPIRED = "key_expired"
    KEY_EXPIRING = "key_expiring"
    KEY_REVOKED = "key_revoked"
    OWNER_LEFT = "owner_left"
    SITE_SILENT = "site_silent"

    def __str__(self) -> str:
        return str(self.value)
