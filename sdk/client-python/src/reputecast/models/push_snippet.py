from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushSnippet")



@_attrs_define
class PushSnippet:
    """ 
        Attributes:
            import_url (str):
            script (str):
            worker_bytes (int):
            worker_file (str):
            worker_url (str):
            worker_version (int):
     """

    import_url: str
    script: str
    worker_bytes: int
    worker_file: str
    worker_url: str
    worker_version: int





    def to_dict(self) -> dict[str, Any]:
        import_url = self.import_url

        script = self.script

        worker_bytes = self.worker_bytes

        worker_file = self.worker_file

        worker_url = self.worker_url

        worker_version = self.worker_version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "importUrl": import_url,
            "script": script,
            "workerBytes": worker_bytes,
            "workerFile": worker_file,
            "workerUrl": worker_url,
            "workerVersion": worker_version,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        import_url = d.pop("importUrl")

        script = d.pop("script")

        worker_bytes = d.pop("workerBytes")

        worker_file = d.pop("workerFile")

        worker_url = d.pop("workerUrl")

        worker_version = d.pop("workerVersion")

        push_snippet = cls(
            import_url=import_url,
            script=script,
            worker_bytes=worker_bytes,
            worker_file=worker_file,
            worker_url=worker_url,
            worker_version=worker_version,
        )

        return push_snippet

