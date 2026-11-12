from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_schedule_mode import PushScheduleMode
from ..models.push_schedule_outside import PushScheduleOutside
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PushSchedule")



@_attrs_define
class PushSchedule:
    """ 
        Attributes:
            mode (PushScheduleMode):
            outside (PushScheduleOutside):
            ttl_minutes (int):
            zone (str):
            send_at (datetime.datetime | Unset):
     """

    mode: PushScheduleMode
    outside: PushScheduleOutside
    ttl_minutes: int
    zone: str
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        outside = self.outside.value

        ttl_minutes = self.ttl_minutes

        zone = self.zone

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
            "outside": outside,
            "ttlMinutes": ttl_minutes,
            "zone": zone,
        })
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = PushScheduleMode(d.pop("mode"))




        outside = PushScheduleOutside(d.pop("outside"))




        ttl_minutes = d.pop("ttlMinutes")

        zone = d.pop("zone")

        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        push_schedule = cls(
            mode=mode,
            outside=outside,
            ttl_minutes=ttl_minutes,
            zone=zone,
            send_at=send_at,
        )

        return push_schedule

