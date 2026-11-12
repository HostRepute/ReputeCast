from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="AutomationPathCount")



@_attrs_define
class AutomationPathCount:
    """ 
        Attributes:
            count (int):
            path (str):  Example: yes.
     """

    count: int
    path: str





    def to_dict(self) -> dict[str, Any]:
        count = self.count

        path = self.path


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "count": count,
            "path": path,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        count = d.pop("count")

        path = d.pop("path")

        automation_path_count = cls(
            count=count,
            path=path,
        )

        return automation_path_count

