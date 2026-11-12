from enum import StrEnum

class InboundRouteViewStatus(StrEnum):
    FAILING = "failing"
    NOT_SET_UP = "not_set_up"
    PENDING = "pending"
    RECEIVING = "receiving"

    def __str__(self) -> str:
        return str(self.value)
