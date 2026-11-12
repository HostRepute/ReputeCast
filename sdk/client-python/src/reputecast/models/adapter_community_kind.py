from enum import StrEnum

class AdapterCommunityKind(StrEnum):
    HTTP = "http"
    REMOTE = "remote"
    WASM = "wasm"

    def __str__(self) -> str:
        return str(self.value)
