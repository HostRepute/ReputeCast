from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralQualityLine")



@_attrs_define
class ReferralQualityLine:
    """ 
        Attributes:
            others (float):
            others_n (int):
            others_of (int):
            referred (float):
            referred_n (int):
            referred_of (int):
     """

    others: float
    others_n: int
    others_of: int
    referred: float
    referred_n: int
    referred_of: int





    def to_dict(self) -> dict[str, Any]:
        others = self.others

        others_n = self.others_n

        others_of = self.others_of

        referred = self.referred

        referred_n = self.referred_n

        referred_of = self.referred_of


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "others": others,
            "othersN": others_n,
            "othersOf": others_of,
            "referred": referred,
            "referredN": referred_n,
            "referredOf": referred_of,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        others = d.pop("others")

        others_n = d.pop("othersN")

        others_of = d.pop("othersOf")

        referred = d.pop("referred")

        referred_n = d.pop("referredN")

        referred_of = d.pop("referredOf")

        referral_quality_line = cls(
            others=others,
            others_n=others_n,
            others_of=others_of,
            referred=referred,
            referred_n=referred_n,
            referred_of=referred_of,
        )

        return referral_quality_line

