from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ReferralCouponOffer")



@_attrs_define
class ReferralCouponOffer:
    """ 
        Attributes:
            applies_to (str):
            discount (str):
            id (str):
            issued (int):
            name (str):
            redeemed (int):
     """

    applies_to: str
    discount: str
    id: str
    issued: int
    name: str
    redeemed: int





    def to_dict(self) -> dict[str, Any]:
        applies_to = self.applies_to

        discount = self.discount

        id = self.id

        issued = self.issued

        name = self.name

        redeemed = self.redeemed


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "appliesTo": applies_to,
            "discount": discount,
            "id": id,
            "issued": issued,
            "name": name,
            "redeemed": redeemed,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        applies_to = d.pop("appliesTo")

        discount = d.pop("discount")

        id = d.pop("id")

        issued = d.pop("issued")

        name = d.pop("name")

        redeemed = d.pop("redeemed")

        referral_coupon_offer = cls(
            applies_to=applies_to,
            discount=discount,
            id=id,
            issued=issued,
            name=name,
            redeemed=redeemed,
        )

        return referral_coupon_offer

