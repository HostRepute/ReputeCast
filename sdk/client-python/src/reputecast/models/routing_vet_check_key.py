from enum import StrEnum

class RoutingVetCheckKey(StrEnum):
    AVAILABLE = "available"
    BULK_TERMS = "bulk_terms"
    DOMAIN = "domain"
    QUOTA = "quota"
    SANDBOX = "sandbox"
    WARM = "warm"

    def __str__(self) -> str:
        return str(self.value)
