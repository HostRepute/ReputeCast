from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReportSuppressResult")



@_attrs_define
class ReportSuppressResult:
    """ 
        Attributes:
            added (int):
            already (int): Already on the list
     """

    added: int
    already: int





    def to_dict(self) -> dict[str, Any]:
        added = self.added

        already = self.already


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "added": added,
            "already": already,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        added = d.pop("added")

        already = d.pop("already")

        report_suppress_result = cls(
            added=added,
            already=already,
        )

        return report_suppress_result

