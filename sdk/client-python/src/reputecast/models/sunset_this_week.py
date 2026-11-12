from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="SunsetThisWeek")



@_attrs_define
class SunsetThisWeek:
    """ 
        Attributes:
            entering (int): Contacts entering Sunset
            stopping (int): Contacts whose Broadcasts stop
            enter_day (str | Unset):
            stop_day (str | Unset):
     """

    entering: int
    stopping: int
    enter_day: str | Unset = UNSET
    stop_day: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        entering = self.entering

        stopping = self.stopping

        enter_day = self.enter_day

        stop_day = self.stop_day


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "entering": entering,
            "stopping": stopping,
        })
        if enter_day is not UNSET:
            field_dict["enterDay"] = enter_day
        if stop_day is not UNSET:
            field_dict["stopDay"] = stop_day

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        entering = d.pop("entering")

        stopping = d.pop("stopping")

        enter_day = d.pop("enterDay", UNSET)

        stop_day = d.pop("stopDay", UNSET)

        sunset_this_week = cls(
            entering=entering,
            stopping=stopping,
            enter_day=enter_day,
            stop_day=stop_day,
        )

        return sunset_this_week

