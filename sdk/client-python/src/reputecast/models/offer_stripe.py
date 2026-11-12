from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="OfferStripe")



@_attrs_define
class OfferStripe:
    """ 
        Attributes:
            missing (bool):
            coupon (str | Unset):
            coupon_url (str | Unset):
            mode (str | Unset):
            promotion_code (str | Unset):
     """

    missing: bool
    coupon: str | Unset = UNSET
    coupon_url: str | Unset = UNSET
    mode: str | Unset = UNSET
    promotion_code: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        missing = self.missing

        coupon = self.coupon

        coupon_url = self.coupon_url

        mode = self.mode

        promotion_code = self.promotion_code


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "missing": missing,
        })
        if coupon is not UNSET:
            field_dict["coupon"] = coupon
        if coupon_url is not UNSET:
            field_dict["couponUrl"] = coupon_url
        if mode is not UNSET:
            field_dict["mode"] = mode
        if promotion_code is not UNSET:
            field_dict["promotionCode"] = promotion_code

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        missing = d.pop("missing")

        coupon = d.pop("coupon", UNSET)

        coupon_url = d.pop("couponUrl", UNSET)

        mode = d.pop("mode", UNSET)

        promotion_code = d.pop("promotionCode", UNSET)

        offer_stripe = cls(
            missing=missing,
            coupon=coupon,
            coupon_url=coupon_url,
            mode=mode,
            promotion_code=promotion_code,
        )

        return offer_stripe

