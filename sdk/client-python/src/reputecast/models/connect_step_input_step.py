from enum import StrEnum

class ConnectStepInputStep(StrEnum):
    CONNECTION = "connection"
    CONTACTS = "contacts"
    DOMAIN = "domain"
    STREAMS = "streams"
    TEST = "test"

    def __str__(self) -> str:
        return str(self.value)
