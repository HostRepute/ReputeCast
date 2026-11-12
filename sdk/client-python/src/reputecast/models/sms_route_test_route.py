from enum import StrEnum

class SmsRouteTestRoute(StrEnum):
    NONE = "none"
    OTHER = "other"
    ROUTE = "route"

    def __str__(self) -> str:
        return str(self.value)
