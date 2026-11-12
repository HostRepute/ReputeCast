from enum import StrEnum

class PushContactSubscriptionLinkedBy(StrEnum):
    ACCOUNT = "account"
    LINK = "link"
    MEMBER = "member"
    PREFERENCES = "preferences"
    VALUE_4 = ""

    def __str__(self) -> str:
        return str(self.value)
