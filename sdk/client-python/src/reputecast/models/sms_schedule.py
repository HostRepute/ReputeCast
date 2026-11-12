from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_schedule_mode import SmsScheduleMode
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sms_zone_group import SmsZoneGroup





T = TypeVar("T", bound="SmsSchedule")



@_attrs_define
class SmsSchedule:
    """ 
        Attributes:
            groups (list[SmsZoneGroup]):
            mode (SmsScheduleMode):
            now (list[SmsZoneGroup]):
            window (str):
            window_from (int):
            window_to (int):
            end (datetime.datetime | Unset):
            local_date (str | Unset):
            local_time (str | Unset):
            send_at (datetime.datetime | Unset):
     """

    groups: list[SmsZoneGroup]
    mode: SmsScheduleMode
    now: list[SmsZoneGroup]
    window: str
    window_from: int
    window_to: int
    end: datetime.datetime | Unset = UNSET
    local_date: str | Unset = UNSET
    local_time: str | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_zone_group import SmsZoneGroup # noqa: PLC0415
        groups = []
        for groups_item_data in self.groups:
            groups_item = groups_item_data.to_dict()
            groups.append(groups_item)



        mode = self.mode.value

        now = []
        for now_item_data in self.now:
            now_item = now_item_data.to_dict()
            now.append(now_item)



        window = self.window

        window_from = self.window_from

        window_to = self.window_to

        end: str | Unset = UNSET
        if not isinstance(self.end, Unset):
            end = self.end.isoformat()

        local_date = self.local_date

        local_time = self.local_time

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "groups": groups,
            "mode": mode,
            "now": now,
            "window": window,
            "windowFrom": window_from,
            "windowTo": window_to,
        })
        if end is not UNSET:
            field_dict["end"] = end
        if local_date is not UNSET:
            field_dict["localDate"] = local_date
        if local_time is not UNSET:
            field_dict["localTime"] = local_time
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_zone_group import SmsZoneGroup # noqa: PLC0415
        d = dict(src_dict)
        groups = []
        _groups = d.pop("groups")
        for groups_item_data in (_groups):
            groups_item = SmsZoneGroup.from_dict(groups_item_data)



            groups.append(groups_item)


        mode = SmsScheduleMode(d.pop("mode"))




        now = []
        _now = d.pop("now")
        for now_item_data in (_now):
            now_item = SmsZoneGroup.from_dict(now_item_data)



            now.append(now_item)


        window = d.pop("window")

        window_from = d.pop("windowFrom")

        window_to = d.pop("windowTo")

        _end = d.pop("end", UNSET)
        end: datetime.datetime | Unset
        if isinstance(_end,  Unset):
            end = UNSET
        else:
            end = datetime.datetime.fromisoformat(_end)




        local_date = d.pop("localDate", UNSET)

        local_time = d.pop("localTime", UNSET)

        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        sms_schedule = cls(
            groups=groups,
            mode=mode,
            now=now,
            window=window,
            window_from=window_from,
            window_to=window_to,
            end=end,
            local_date=local_date,
            local_time=local_time,
            send_at=send_at,
        )

        return sms_schedule

