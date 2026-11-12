from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.digest_schedule_frequency import DigestScheduleFrequency
from ..models.digest_schedule_month_rule import DigestScheduleMonthRule
from ..types import UNSET, Unset






T = TypeVar("T", bound="DigestSchedule")



@_attrs_define
class DigestSchedule:
    """ 
        Attributes:
            frequency (DigestScheduleFrequency):
            time (str):  Example: 08:00.
            timezone (str):  Example: America/New_York.
            weekday (int):
            count (int | Unset):
            month_day (int | Unset):
            month_rule (DigestScheduleMonthRule | Unset):
            weekdays_only (bool | Unset):
            window_from (str | Unset):
            window_until (str | Unset):
     """

    frequency: DigestScheduleFrequency
    time: str
    timezone: str
    weekday: int
    count: int | Unset = UNSET
    month_day: int | Unset = UNSET
    month_rule: DigestScheduleMonthRule | Unset = UNSET
    weekdays_only: bool | Unset = UNSET
    window_from: str | Unset = UNSET
    window_until: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        frequency = self.frequency.value

        time = self.time

        timezone = self.timezone

        weekday = self.weekday

        count = self.count

        month_day = self.month_day

        month_rule: str | Unset = UNSET
        if not isinstance(self.month_rule, Unset):
            month_rule = self.month_rule.value


        weekdays_only = self.weekdays_only

        window_from = self.window_from

        window_until = self.window_until


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "frequency": frequency,
            "time": time,
            "timezone": timezone,
            "weekday": weekday,
        })
        if count is not UNSET:
            field_dict["count"] = count
        if month_day is not UNSET:
            field_dict["monthDay"] = month_day
        if month_rule is not UNSET:
            field_dict["monthRule"] = month_rule
        if weekdays_only is not UNSET:
            field_dict["weekdaysOnly"] = weekdays_only
        if window_from is not UNSET:
            field_dict["windowFrom"] = window_from
        if window_until is not UNSET:
            field_dict["windowUntil"] = window_until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        frequency = DigestScheduleFrequency(d.pop("frequency"))




        time = d.pop("time")

        timezone = d.pop("timezone")

        weekday = d.pop("weekday")

        count = d.pop("count", UNSET)

        month_day = d.pop("monthDay", UNSET)

        _month_rule = d.pop("monthRule", UNSET)
        month_rule: DigestScheduleMonthRule | Unset
        if isinstance(_month_rule,  Unset):
            month_rule = UNSET
        else:
            month_rule = DigestScheduleMonthRule(_month_rule)




        weekdays_only = d.pop("weekdaysOnly", UNSET)

        window_from = d.pop("windowFrom", UNSET)

        window_until = d.pop("windowUntil", UNSET)

        digest_schedule = cls(
            frequency=frequency,
            time=time,
            timezone=timezone,
            weekday=weekday,
            count=count,
            month_day=month_day,
            month_rule=month_rule,
            weekdays_only=weekdays_only,
            window_from=window_from,
            window_until=window_until,
        )

        return digest_schedule

