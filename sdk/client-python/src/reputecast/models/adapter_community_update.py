from enum import StrEnum

class AdapterCommunityUpdate(StrEnum):
    UNMAINTAINED = "unmaintained"
    UPDATE_AVAILABLE = "update_available"
    UP_TO_DATE = "up_to_date"

    def __str__(self) -> str:
        return str(self.value)
