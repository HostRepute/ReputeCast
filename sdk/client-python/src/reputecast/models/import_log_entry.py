from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.import_log_entry_kind import ImportLogEntryKind
from typing import cast
import datetime






T = TypeVar("T", bound="ImportLogEntry")



@_attrs_define
class ImportLogEntry:
    """ 
        Attributes:
            at (datetime.datetime):
            kind (ImportLogEntryKind):
            message (str):
     """

    at: datetime.datetime
    kind: ImportLogEntryKind
    message: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        kind = self.kind.value

        message = self.message


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "kind": kind,
            "message": message,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        kind = ImportLogEntryKind(d.pop("kind"))




        message = d.pop("message")

        import_log_entry = cls(
            at=at,
            kind=kind,
            message=message,
        )

        return import_log_entry

