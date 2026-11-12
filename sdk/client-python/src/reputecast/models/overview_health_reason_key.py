from enum import StrEnum

class OverviewHealthReasonKey(StrEnum):
    AUTH = "auth"
    BLOCKLIST_DOMAIN = "blocklist_domain"
    BLOCKLIST_IP = "blocklist_ip"
    CHECKS = "checks"
    PROVIDER_STATUS = "provider_status"
    RATES = "rates"
    WARMUP = "warmup"

    def __str__(self) -> str:
        return str(self.value)
