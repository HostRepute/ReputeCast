from enum import StrEnum

class ProviderLimitRateReason(StrEnum):
    NOT_REPORTED = "not_reported"
    NO_DATA = "no_data"

    def __str__(self) -> str:
        return str(self.value)
