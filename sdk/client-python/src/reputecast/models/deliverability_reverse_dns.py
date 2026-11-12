from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="DeliverabilityReverseDNS")



@_attrs_define
class DeliverabilityReverseDNS:
    """ 
        Attributes:
            matches (bool):
            status (str):
            ptr (str | Unset):
     """

    matches: bool
    status: str
    ptr: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        matches = self.matches

        status = self.status

        ptr = self.ptr


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "matches": matches,
            "status": status,
        })
        if ptr is not UNSET:
            field_dict["ptr"] = ptr

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        matches = d.pop("matches")

        status = d.pop("status")

        ptr = d.pop("ptr", UNSET)

        deliverability_reverse_dns = cls(
            matches=matches,
            status=status,
            ptr=ptr,
        )

        return deliverability_reverse_dns

