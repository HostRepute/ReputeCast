from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="PushZone")



@_attrs_define
class PushZone:
    """ 
        Attributes:
            inside (bool):
            local_time (str):
            subscribers (int):
            until (datetime.datetime):
            zone (str):
     """

    inside: bool
    local_time: str
    subscribers: int
    until: datetime.datetime
    zone: str





    def to_dict(self) -> dict[str, Any]:
        inside = self.inside

        local_time = self.local_time

        subscribers = self.subscribers

        until = self.until.isoformat()

        zone = self.zone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "inside": inside,
            "localTime": local_time,
            "subscribers": subscribers,
            "until": until,
            "zone": zone,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        inside = d.pop("inside")

        local_time = d.pop("localTime")

        subscribers = d.pop("subscribers")

        until = datetime.datetime.fromisoformat(d.pop("until"))




        zone = d.pop("zone")

        push_zone = cls(
            inside=inside,
            local_time=local_time,
            subscribers=subscribers,
            until=until,
            zone=zone,
        )

        return push_zone

