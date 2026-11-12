from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.readiness_schedule_mode import ReadinessScheduleMode
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ReadinessSchedule")



@_attrs_define
class ReadinessSchedule:
    """ 
        Attributes:
            mode (ReadinessScheduleMode):
            at (datetime.datetime | Unset):
            date (str | Unset):
            time (str | Unset):
            timezone (str | Unset):
     """

    mode: ReadinessScheduleMode
    at: datetime.datetime | Unset = UNSET
    date: str | Unset = UNSET
    time: str | Unset = UNSET
    timezone: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        date = self.date

        time = self.time

        timezone = self.timezone


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if date is not UNSET:
            field_dict["date"] = date
        if time is not UNSET:
            field_dict["time"] = time
        if timezone is not UNSET:
            field_dict["timezone"] = timezone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = ReadinessScheduleMode(d.pop("mode"))




        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        date = d.pop("date", UNSET)

        time = d.pop("time", UNSET)

        timezone = d.pop("timezone", UNSET)

        readiness_schedule = cls(
            mode=mode,
            at=at,
            date=date,
            time=time,
            timezone=timezone,
        )

        return readiness_schedule

