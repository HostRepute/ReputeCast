from enum import StrEnum

class RoutingConditionViewField(StrEnum):
    CATEGORY = "category"
    LIST = "list"
    MAILBOX_PROVIDER = "mailbox_provider"
    SENDER_DOMAIN = "sender_domain"

    def __str__(self) -> str:
        return str(self.value)
