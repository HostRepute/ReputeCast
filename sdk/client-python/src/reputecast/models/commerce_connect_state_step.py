from enum import StrEnum

class CommerceConnectStateStep(StrEnum):
    ADDRESS = "address"
    APP = "app"
    DONE = "done"
    INSTALL = "install"
    KEY = "key"
    KEYS = "keys"
    PLUGIN = "plugin"
    SYNC = "sync"
    WEBHOOKS = "webhooks"

    def __str__(self) -> str:
        return str(self.value)
