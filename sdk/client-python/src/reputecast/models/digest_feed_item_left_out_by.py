from enum import StrEnum

class DigestFeedItemLeftOutBy(StrEnum):
    AGE = "age"
    CATEGORY = "category"
    TAG = "tag"
    VALUE_3 = ""

    def __str__(self) -> str:
        return str(self.value)
