from enum import StrEnum

class PageSettingsVisibility(StrEnum):
    MEMBERS = "members"
    PAID = "paid"
    PUBLIC = "public"

    def __str__(self) -> str:
        return str(self.value)
