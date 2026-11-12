from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="TrackerRefusal")



@_attrs_define
class TrackerRefusal:
    """ 
        Attributes:
            count (int):
            last_at (datetime.datetime):
            origin (str):
     """

    count: int
    last_at: datetime.datetime
    origin: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        last_at = self.last_at.isoformat()

        origin = self.origin


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "lastAt": last_at,
            "origin": origin,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        last_at = datetime.datetime.fromisoformat(d.pop("lastAt"))




        origin = d.pop("origin")

        tracker_refusal = cls(
            count=count,
            last_at=last_at,
            origin=origin,
        )

        return tracker_refusal

