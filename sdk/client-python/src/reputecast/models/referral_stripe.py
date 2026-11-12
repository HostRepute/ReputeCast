from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.referral_coupon_offer import ReferralCouponOffer





T = TypeVar("T", bound="ReferralStripe")



@_attrs_define
class ReferralStripe:
    """ 
        Attributes:
            account (str):
            connected (bool):
            offers (list[ReferralCouponOffer]):
     """

    account: str
    connected: bool
    offers: list[ReferralCouponOffer]





    def to_dict(self) -> dict[str, Any]:
        from ..models.referral_coupon_offer import ReferralCouponOffer # noqa: PLC0415
        account = self.account

        connected = self.connected

        offers = []
        for offers_item_data in self.offers:
            offers_item = offers_item_data.to_dict()
            offers.append(offers_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "account": account,
            "connected": connected,
            "offers": offers,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.referral_coupon_offer import ReferralCouponOffer # noqa: PLC0415
        d = dict(src_dict)
        account = d.pop("account")

        connected = d.pop("connected")

        offers = []
        _offers = d.pop("offers")
        for offers_item_data in (_offers):
            offers_item = ReferralCouponOffer.from_dict(offers_item_data)



            offers.append(offers_item)


        referral_stripe = cls(
            account=account,
            connected=connected,
            offers=offers,
        )

        return referral_stripe

