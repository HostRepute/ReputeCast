from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_price_origin import EarnPriceOrigin
from ..models.earn_price_period import EarnPricePeriod
from ..models.earn_price_status import EarnPriceStatus
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.earn_price_offer import EarnPriceOffer





T = TypeVar("T", bound="EarnPrice")



@_attrs_define
class EarnPrice:
    """ 
        Attributes:
            amount_cents (int):
            currency (str):
            discounted (int):
            id (str):
            members (int):
            offers (list[EarnPriceOffer]):
            origin (EarnPriceOrigin):
            paying (int):
            period (EarnPricePeriod):
            status (EarnPriceStatus):
            stripe_id (str | Unset):
            stripe_live_id (str | Unset):
            stripe_test_id (str | Unset):
            stripe_url (str | Unset):
     """

    amount_cents: int
    currency: str
    discounted: int
    id: str
    members: int
    offers: list[EarnPriceOffer]
    origin: EarnPriceOrigin
    paying: int
    period: EarnPricePeriod
    status: EarnPriceStatus
    stripe_id: str | Unset = UNSET
    stripe_live_id: str | Unset = UNSET
    stripe_test_id: str | Unset = UNSET
    stripe_url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_price_offer import EarnPriceOffer # noqa: PLC0415
        amount_cents = self.amount_cents

        currency = self.currency

        discounted = self.discounted

        id = self.id

        members = self.members

        offers = []
        for offers_item_data in self.offers:
            offers_item = offers_item_data.to_dict()
            offers.append(offers_item)



        origin = self.origin.value

        paying = self.paying

        period = self.period.value

        status = self.status.value

        stripe_id = self.stripe_id

        stripe_live_id = self.stripe_live_id

        stripe_test_id = self.stripe_test_id

        stripe_url = self.stripe_url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "currency": currency,
            "discounted": discounted,
            "id": id,
            "members": members,
            "offers": offers,
            "origin": origin,
            "paying": paying,
            "period": period,
            "status": status,
        })
        if stripe_id is not UNSET:
            field_dict["stripeId"] = stripe_id
        if stripe_live_id is not UNSET:
            field_dict["stripeLiveId"] = stripe_live_id
        if stripe_test_id is not UNSET:
            field_dict["stripeTestId"] = stripe_test_id
        if stripe_url is not UNSET:
            field_dict["stripeUrl"] = stripe_url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_price_offer import EarnPriceOffer # noqa: PLC0415
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        currency = d.pop("currency")

        discounted = d.pop("discounted")

        id = d.pop("id")

        members = d.pop("members")

        offers = []
        _offers = d.pop("offers")
        for offers_item_data in (_offers):
            offers_item = EarnPriceOffer.from_dict(offers_item_data)



            offers.append(offers_item)


        origin = EarnPriceOrigin(d.pop("origin"))




        paying = d.pop("paying")

        period = EarnPricePeriod(d.pop("period"))




        status = EarnPriceStatus(d.pop("status"))




        stripe_id = d.pop("stripeId", UNSET)

        stripe_live_id = d.pop("stripeLiveId", UNSET)

        stripe_test_id = d.pop("stripeTestId", UNSET)

        stripe_url = d.pop("stripeUrl", UNSET)

        earn_price = cls(
            amount_cents=amount_cents,
            currency=currency,
            discounted=discounted,
            id=id,
            members=members,
            offers=offers,
            origin=origin,
            paying=paying,
            period=period,
            status=status,
            stripe_id=stripe_id,
            stripe_live_id=stripe_live_id,
            stripe_test_id=stripe_test_id,
            stripe_url=stripe_url,
        )

        return earn_price

