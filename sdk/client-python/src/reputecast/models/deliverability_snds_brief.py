from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="DeliverabilitySNDSBrief")



@_attrs_define
class DeliverabilitySNDSBrief:
    """ 
        Attributes:
            filter_result (str):
            trap_hits (int):
     """

    filter_result: str
    trap_hits: int





    def to_dict(self) -> dict[str, Any]:
        filter_result = self.filter_result

        trap_hits = self.trap_hits


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "filterResult": filter_result,
            "trapHits": trap_hits,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        filter_result = d.pop("filterResult")

        trap_hits = d.pop("trapHits")

        deliverability_snds_brief = cls(
            filter_result=filter_result,
            trap_hits=trap_hits,
        )

        return deliverability_snds_brief

