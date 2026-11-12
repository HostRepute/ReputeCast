from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ApprovalAudienceRule")



@_attrs_define
class ApprovalAudienceRule:
    """ 
        Attributes:
            enabled (bool):
            n (int):
     """

    enabled: bool
    n: int





    def to_dict(self) -> dict[str, Any]:
        enabled = self.enabled

        n = self.n


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "enabled": enabled,
            "n": n,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        enabled = d.pop("enabled")

        n = d.pop("n")

        approval_audience_rule = cls(
            enabled=enabled,
            n=n,
        )

        return approval_audience_rule

