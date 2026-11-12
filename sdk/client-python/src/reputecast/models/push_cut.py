from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.push_cut_platform import PushCutPlatform






T = TypeVar("T", bound="PushCut")



@_attrs_define
class PushCut:
    """ 
        Attributes:
            lines (int):
            platform (PushCutPlatform):
            title (bool):
     """

    lines: int
    platform: PushCutPlatform
    title: bool





    def to_dict(self) -> dict[str, Any]:
        lines = self.lines

        platform = self.platform.value

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "lines": lines,
            "platform": platform,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        lines = d.pop("lines")

        platform = PushCutPlatform(d.pop("platform"))




        title = d.pop("title")

        push_cut = cls(
            lines=lines,
            platform=platform,
            title=title,
        )

        return push_cut

