from enum import StrEnum

class ListSmsSuppressionsReason(StrEnum):
    FILTERED = "filtered"
    INVALID = "invalid"
    LANDLINE = "landline"
    MANUAL = "manual"
    NOT_IN_SERVICE = "not_in_service"
    OPTED_OUT = "opted_out"
    PROVIDER_LIST = "provider_list"

    def __str__(self) -> str:
        return str(self.value)
