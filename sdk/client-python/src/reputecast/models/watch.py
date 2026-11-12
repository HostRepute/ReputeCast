from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.watch_hours import WatchHours






T = TypeVar("T", bound="Watch")



@_attrs_define
class Watch:
    """ 
        Attributes:
            alert (bool):
            enabled (bool):
            hours (WatchHours):
            postmaster (bool):
     """

    alert: bool
    enabled: bool
    hours: WatchHours
    postmaster: bool





    def to_dict(self) -> dict[str, Any]:
        alert = self.alert

        enabled = self.enabled

        hours = self.hours.value

        postmaster = self.postmaster


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alert": alert,
            "enabled": enabled,
            "hours": hours,
            "postmaster": postmaster,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        alert = d.pop("alert")

        enabled = d.pop("enabled")

        hours = WatchHours(d.pop("hours"))




        postmaster = d.pop("postmaster")

        watch = cls(
            alert=alert,
            enabled=enabled,
            hours=hours,
            postmaster=postmaster,
        )

        return watch

