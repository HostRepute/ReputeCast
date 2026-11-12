from enum import StrEnum

class RoutingConditionField(StrEnum):
    CATEGORY = "category"
    LIST = "list"
    MAILBOX_PROVIDER = "mailbox_provider"
    SENDER_DOMAIN = "sender_domain"

    def __str__(self) -> str:
        return str(self.value)
