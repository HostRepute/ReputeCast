from enum import StrEnum

class InboundRouteViewStatusReason(StrEnum):
    DROPPED_REPORTS = "dropped_reports"
    MAILBOX_AUTH = "mailbox_auth"
    MAILBOX_UNREACHABLE = "mailbox_unreachable"
    MX = "mx"
    REGISTER = "register"

    def __str__(self) -> str:
        return str(self.value)
