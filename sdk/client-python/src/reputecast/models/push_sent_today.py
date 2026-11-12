from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_sent_today_kind import PushSentTodayKind
from typing import cast
import datetime






T = TypeVar("T", bound="PushSentToday")



@_attrs_define
class PushSentToday:
    """ 
        Attributes:
            at (datetime.datetime):
            by (str):
            kind (PushSentTodayKind):
            name (str):
            to (int):
     """

    at: datetime.datetime
    by: str
    kind: PushSentTodayKind
    name: str
    to: int





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        by = self.by

        kind = self.kind.value

        name = self.name

        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "by": by,
            "kind": kind,
            "name": name,
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        by = d.pop("by")

        kind = PushSentTodayKind(d.pop("kind"))




        name = d.pop("name")

        to = d.pop("to")

        push_sent_today = cls(
            at=at,
            by=by,
            kind=kind,
            name=name,
            to=to,
        )

        return push_sent_today

