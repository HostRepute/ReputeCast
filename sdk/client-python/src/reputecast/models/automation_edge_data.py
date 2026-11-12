from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationEdgeData")



@_attrs_define
class AutomationEdgeData:
    """ 
        Attributes:
            path (str | Unset):
     """

    path: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        path = self.path


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if path is not UNSET:
            field_dict["path"] = path

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        path = d.pop("path", UNSET)

        automation_edge_data = cls(
            path=path,
        )

        return automation_edge_data

