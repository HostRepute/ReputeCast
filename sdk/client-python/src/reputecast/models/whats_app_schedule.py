from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_schedule_mode import WhatsAppScheduleMode
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppSchedule")



@_attrs_define
class WhatsAppSchedule:
    """ 
        Attributes:
            label (str):
            mode (WhatsAppScheduleMode):
            local_date (str | Unset):
            local_minute (int | Unset):
            send_at (datetime.datetime | Unset):
     """

    label: str
    mode: WhatsAppScheduleMode
    local_date: str | Unset = UNSET
    local_minute: int | Unset = UNSET
    send_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        label = self.label

        mode = self.mode.value

        local_date = self.local_date

        local_minute = self.local_minute

        send_at: str | Unset = UNSET
        if not isinstance(self.send_at, Unset):
            send_at = self.send_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "label": label,
            "mode": mode,
        })
        if local_date is not UNSET:
            field_dict["localDate"] = local_date
        if local_minute is not UNSET:
            field_dict["localMinute"] = local_minute
        if send_at is not UNSET:
            field_dict["sendAt"] = send_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        label = d.pop("label")

        mode = WhatsAppScheduleMode(d.pop("mode"))




        local_date = d.pop("localDate", UNSET)

        local_minute = d.pop("localMinute", UNSET)

        _send_at = d.pop("sendAt", UNSET)
        send_at: datetime.datetime | Unset
        if isinstance(_send_at,  Unset):
            send_at = UNSET
        else:
            send_at = datetime.datetime.fromisoformat(_send_at)




        whats_app_schedule = cls(
            label=label,
            mode=mode,
            local_date=local_date,
            local_minute=local_minute,
            send_at=send_at,
        )

        return whats_app_schedule

