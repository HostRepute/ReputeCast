from enum import StrEnum

class AutomationConditionKind(StrEnum):
    BOUGHT_PRODUCT = "bought_product"
    CLICKED = "clicked"
    EVENT = "event"
    FIELD = "field"
    LIST = "list"
    OPENED = "opened"
    PLACED_ORDER = "placed_order"
    SMS_CLICKED = "sms_clicked"
    SMS_CONSENT = "sms_consent"
    SMS_REPLIED = "sms_replied"
    TAG = "tag"

    def __str__(self) -> str:
        return str(self.value)
