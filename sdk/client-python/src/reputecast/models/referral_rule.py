from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralRule")



@_attrs_define
class ReferralRule:
    """ 
        Attributes:
            on (bool):
            threshold (int):
     """

    on: bool
    threshold: int





    def to_dict(self) -> dict[str, Any]:
        on = self.on

        threshold = self.threshold


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "on": on,
            "threshold": threshold,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        on = d.pop("on")

        threshold = d.pop("threshold")

        referral_rule = cls(
            on=on,
            threshold=threshold,
        )

        return referral_rule

