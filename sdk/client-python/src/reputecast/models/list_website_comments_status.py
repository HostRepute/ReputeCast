from enum import StrEnum

class ListWebsiteCommentsStatus(StrEnum):
    HIDDEN = "hidden"
    PENDING = "pending"
    PUBLISHED = "published"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
