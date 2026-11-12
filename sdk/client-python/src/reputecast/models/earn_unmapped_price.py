from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_unmapped_price_mode import EarnUnmappedPriceMode
from ..models.earn_unmapped_price_period import EarnUnmappedPricePeriod
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_member_row import EarnMemberRow





T = TypeVar("T", bound="EarnUnmappedPrice")



@_attrs_define
class EarnUnmappedPrice:
    """ 
        Attributes:
            amount_cents (int):
            currency (str):
            found_at (datetime.datetime):
            id (str):
            members (list[EarnMemberRow]):
            mode (EarnUnmappedPriceMode):
            period (EarnUnmappedPricePeriod):
            price_id (str):
            subscriptions (int):
     """

    amount_cents: int
    currency: str
    found_at: datetime.datetime
    id: str
    members: list[EarnMemberRow]
    mode: EarnUnmappedPriceMode
    period: EarnUnmappedPricePeriod
    price_id: str
    subscriptions: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_member_row import EarnMemberRow # noqa: PLC0415
        amount_cents = self.amount_cents

        currency = self.currency

        found_at = self.found_at.isoformat()

        id = self.id

        members = []
        for members_item_data in self.members:
            members_item = members_item_data.to_dict()
            members.append(members_item)



        mode = self.mode.value

        period = self.period.value

        price_id = self.price_id

        subscriptions = self.subscriptions


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "currency": currency,
            "foundAt": found_at,
            "id": id,
            "members": members,
            "mode": mode,
            "period": period,
            "priceId": price_id,
            "subscriptions": subscriptions,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_member_row import EarnMemberRow # noqa: PLC0415
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        currency = d.pop("currency")

        found_at = datetime.datetime.fromisoformat(d.pop("foundAt"))




        id = d.pop("id")

        members = []
        _members = d.pop("members")
        for members_item_data in (_members):
            members_item = EarnMemberRow.from_dict(members_item_data)



            members.append(members_item)


        mode = EarnUnmappedPriceMode(d.pop("mode"))




        period = EarnUnmappedPricePeriod(d.pop("period"))




        price_id = d.pop("priceId")

        subscriptions = d.pop("subscriptions")

        earn_unmapped_price = cls(
            amount_cents=amount_cents,
            currency=currency,
            found_at=found_at,
            id=id,
            members=members,
            mode=mode,
            period=period,
            price_id=price_id,
            subscriptions=subscriptions,
        )

        return earn_unmapped_price

