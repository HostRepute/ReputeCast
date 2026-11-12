from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="WarmupLogEntry")



@_attrs_define
class WarmupLogEntry:
    """ 
        Attributes:
            at (datetime.datetime):
            body (str):
            kind (str):
            title (str):
     """

    at: datetime.datetime
    body: str
    kind: str
    title: str





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        body = self.body

        kind = self.kind

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "body": body,
            "kind": kind,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        body = d.pop("body")

        kind = d.pop("kind")

        title = d.pop("title")

        warmup_log_entry = cls(
            at=at,
            body=body,
            kind=kind,
            title=title,
        )

        return warmup_log_entry

