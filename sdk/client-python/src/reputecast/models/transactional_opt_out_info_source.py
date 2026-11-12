from enum import StrEnum

class TransactionalOptOutInfoSource(StrEnum):
    API = "api"
    MEMBER = "member"
    ONE_CLICK = "one_click"
    PREFERENCE_CENTER = "preference_center"
    UNSUBSCRIBE_LINK = "unsubscribe_link"
    VALUE_5 = ""

    def __str__(self) -> str:
        return str(self.value)
