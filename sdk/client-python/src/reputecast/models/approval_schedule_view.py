from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.approval_schedule_view_mode import ApprovalScheduleViewMode
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ApprovalScheduleView")



@_attrs_define
class ApprovalScheduleView:
    """ 
        Attributes:
            auto_pause (str):
            minutes (int):
            mode (ApprovalScheduleViewMode):
            timezone (str):
            at (datetime.datetime | Unset):
     """

    auto_pause: str
    minutes: int
    mode: ApprovalScheduleViewMode
    timezone: str
    at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        auto_pause = self.auto_pause

        minutes = self.minutes

        mode = self.mode.value

        timezone = self.timezone

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "autoPause": auto_pause,
            "minutes": minutes,
            "mode": mode,
            "timezone": timezone,
        })
        if at is not UNSET:
            field_dict["at"] = at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        auto_pause = d.pop("autoPause")

        minutes = d.pop("minutes")

        mode = ApprovalScheduleViewMode(d.pop("mode"))




        timezone = d.pop("timezone")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        approval_schedule_view = cls(
            auto_pause=auto_pause,
            minutes=minutes,
            mode=mode,
            timezone=timezone,
            at=at,
        )

        return approval_schedule_view

