from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralSignalCount")



@_attrs_define
class ReferralSignalCount:
    """ 
        Attributes:
            kind (str):
            referrers (int):
     """

    kind: str
    referrers: int





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind

        referrers = self.referrers


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "referrers": referrers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = d.pop("kind")

        referrers = d.pop("referrers")

        referral_signal_count = cls(
            kind=kind,
            referrers=referrers,
        )

        return referral_signal_count

