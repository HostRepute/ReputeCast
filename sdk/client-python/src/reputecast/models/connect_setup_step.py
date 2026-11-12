from enum import StrEnum

class ConnectSetupStep(StrEnum):
    CONNECTION = "connection"
    CONTACTS = "contacts"
    DOMAIN = "domain"
    DONE = "done"
    STREAMS = "streams"
    TEST = "test"

    def __str__(self) -> str:
        return str(self.value)
