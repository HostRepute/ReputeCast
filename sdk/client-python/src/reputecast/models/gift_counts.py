from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="GiftCounts")



@_attrs_define
class GiftCounts:
    """ 
        Attributes:
            all_ (int):
            expired (int):
            not_redeemed (int):
            redeemed (int):
            refunded (int):
            scheduled (int):
     """

    all_: int
    expired: int
    not_redeemed: int
    redeemed: int
    refunded: int
    scheduled: int





    def to_dict(self) -> dict[str, Any]:
        all_ = self.all_

        expired = self.expired

        not_redeemed = self.not_redeemed

        redeemed = self.redeemed

        refunded = self.refunded

        scheduled = self.scheduled


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "all": all_,
            "expired": expired,
            "notRedeemed": not_redeemed,
            "redeemed": redeemed,
            "refunded": refunded,
            "scheduled": scheduled,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        all_ = d.pop("all")

        expired = d.pop("expired")

        not_redeemed = d.pop("notRedeemed")

        redeemed = d.pop("redeemed")

        refunded = d.pop("refunded")

        scheduled = d.pop("scheduled")

        gift_counts = cls(
            all_=all_,
            expired=expired,
            not_redeemed=not_redeemed,
            redeemed=redeemed,
            refunded=refunded,
            scheduled=scheduled,
        )

        return gift_counts

