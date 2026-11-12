from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationSendWindow")



@_attrs_define
class AutomationSendWindow:
    """ 
        Attributes:
            enabled (bool):
            end (str):
            start (str): HH:MM, recipient-local
     """

    enabled: bool
    end: str
    start: str





    def to_dict(self) -> dict[str, Any]:
        enabled = self.enabled

        end = self.end

        start = self.start


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
            "end": end,
            "start": start,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        enabled = d.pop("enabled")

        end = d.pop("end")

        start = d.pop("start")

        automation_send_window = cls(
            enabled=enabled,
            end=end,
            start=start,
        )

        return automation_send_window

