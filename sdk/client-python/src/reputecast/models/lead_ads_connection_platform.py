from enum import StrEnum

class LeadAdsConnectionPlatform(StrEnum):
    META = "meta"

    def __str__(self) -> str:
        return str(self.value)
