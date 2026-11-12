from enum import StrEnum

class BadgeCapability(StrEnum):
    INBOUND = "inbound"
    OPT_OUT_SYNC = "opt_out_sync"
    PRICING = "pricing"
    RECEIPTS = "receipts"
    REGISTRATION = "registration"
    SENDER_IDS = "sender_ids"
    TOLL_FREE = "toll_free"
    VALUE_1 = "10dlc"

    def __str__(self) -> str:
        return str(self.value)
