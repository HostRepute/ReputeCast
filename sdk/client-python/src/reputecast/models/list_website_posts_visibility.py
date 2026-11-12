from enum import StrEnum

class ListWebsitePostsVisibility(StrEnum):
    MEMBERS = "members"
    PAID = "paid"
    PUBLIC = "public"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
