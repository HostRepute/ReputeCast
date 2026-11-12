from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="PageDiffSetting")



@_attrs_define
class PageDiffSetting:
    """ 
        Attributes:
            after (str):
            before (str):
            key (str):
     """

    after: str
    before: str
    key: str





    def to_dict(self) -> dict[str, Any]:
        after = self.after

        before = self.before

        key = self.key


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "after": after,
            "before": before,
            "key": key,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        after = d.pop("after")

        before = d.pop("before")

        key = d.pop("key")

        page_diff_setting = cls(
            after=after,
            before=before,
            key=key,
        )

        return page_diff_setting

