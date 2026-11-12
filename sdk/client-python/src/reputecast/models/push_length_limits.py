from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PushLengthLimits")



@_attrs_define
class PushLengthLimits:
    """ 
        Attributes:
            message (int):
            title (int):
            title_windows (int):
     """

    message: int
    title: int
    title_windows: int





    def to_dict(self) -> dict[str, Any]:
        message = self.message

        title = self.title

        title_windows = self.title_windows


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "message": message,
            "title": title,
            "titleWindows": title_windows,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        message = d.pop("message")

        title = d.pop("title")

        title_windows = d.pop("titleWindows")

        push_length_limits = cls(
            message=message,
            title=title,
            title_windows=title_windows,
        )

        return push_length_limits

