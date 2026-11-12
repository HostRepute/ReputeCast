from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.revenue_activity_kind import RevenueActivityKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.revenue_activity_data import RevenueActivityData





T = TypeVar("T", bound="RevenueActivity")



@_attrs_define
class RevenueActivity:
    """ 
        Attributes:
            access_paused (bool):
            at (datetime.datetime):
            contact_id (str):
            currency (str):
            data (RevenueActivityData):
            first_name (str):
            kind (RevenueActivityKind):
            name (str):
            period (str):
            price_cents (int):
            source (str):
            source_kind (str):
            tier (str):
            amount_cents (int | Unset):
            canceled_by (str | Unset):
            ends_at (datetime.datetime | Unset):
            offer_cents (int | Unset):
            offer_ends_at (datetime.datetime | Unset):
            saw_offer (str | Unset):
     """

    access_paused: bool
    at: datetime.datetime
    contact_id: str
    currency: str
    data: RevenueActivityData
    first_name: str
    kind: RevenueActivityKind
    name: str
    period: str
    price_cents: int
    source: str
    source_kind: str
    tier: str
    amount_cents: int | Unset = UNSET
    canceled_by: str | Unset = UNSET
    ends_at: datetime.datetime | Unset = UNSET
    offer_cents: int | Unset = UNSET
    offer_ends_at: datetime.datetime | Unset = UNSET
    saw_offer: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.revenue_activity_data import RevenueActivityData # noqa: PLC0415
        access_paused = self.access_paused

        at = self.at.isoformat()

        contact_id = self.contact_id

        currency = self.currency

        data = self.data.to_dict()

        first_name = self.first_name

        kind = self.kind.value

        name = self.name

        period = self.period

        price_cents = self.price_cents

        source = self.source

        source_kind = self.source_kind

        tier = self.tier

        amount_cents = self.amount_cents

        canceled_by = self.canceled_by

        ends_at: str | Unset = UNSET
        if not isinstance(self.ends_at, Unset):
            ends_at = self.ends_at.isoformat()

        offer_cents = self.offer_cents

        offer_ends_at: str | Unset = UNSET
        if not isinstance(self.offer_ends_at, Unset):
            offer_ends_at = self.offer_ends_at.isoformat()

        saw_offer = self.saw_offer


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accessPaused": access_paused,
            "at": at,
            "contactId": contact_id,
            "currency": currency,
            "data": data,
            "firstName": first_name,
            "kind": kind,
            "name": name,
            "period": period,
            "priceCents": price_cents,
            "source": source,
            "sourceKind": source_kind,
            "tier": tier,
        })
        if amount_cents is not UNSET:
            field_dict["amountCents"] = amount_cents
        if canceled_by is not UNSET:
            field_dict["canceledBy"] = canceled_by
        if ends_at is not UNSET:
            field_dict["endsAt"] = ends_at
        if offer_cents is not UNSET:
            field_dict["offerCents"] = offer_cents
        if offer_ends_at is not UNSET:
            field_dict["offerEndsAt"] = offer_ends_at
        if saw_offer is not UNSET:
            field_dict["sawOffer"] = saw_offer

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.revenue_activity_data import RevenueActivityData # noqa: PLC0415
        d = dict(src_dict)
        access_paused = d.pop("accessPaused")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        contact_id = d.pop("contactId")

        currency = d.pop("currency")

        data = RevenueActivityData.from_dict(d.pop("data"))




        first_name = d.pop("firstName")

        kind = RevenueActivityKind(d.pop("kind"))




        name = d.pop("name")

        period = d.pop("period")

        price_cents = d.pop("priceCents")

        source = d.pop("source")

        source_kind = d.pop("sourceKind")

        tier = d.pop("tier")

        amount_cents = d.pop("amountCents", UNSET)

        canceled_by = d.pop("canceledBy", UNSET)

        _ends_at = d.pop("endsAt", UNSET)
        ends_at: datetime.datetime | Unset
        if isinstance(_ends_at,  Unset):
            ends_at = UNSET
        else:
            ends_at = datetime.datetime.fromisoformat(_ends_at)




        offer_cents = d.pop("offerCents", UNSET)

        _offer_ends_at = d.pop("offerEndsAt", UNSET)
        offer_ends_at: datetime.datetime | Unset
        if isinstance(_offer_ends_at,  Unset):
            offer_ends_at = UNSET
        else:
            offer_ends_at = datetime.datetime.fromisoformat(_offer_ends_at)




        saw_offer = d.pop("sawOffer", UNSET)

        revenue_activity = cls(
            access_paused=access_paused,
            at=at,
            contact_id=contact_id,
            currency=currency,
            data=data,
            first_name=first_name,
            kind=kind,
            name=name,
            period=period,
            price_cents=price_cents,
            source=source,
            source_kind=source_kind,
            tier=tier,
            amount_cents=amount_cents,
            canceled_by=canceled_by,
            ends_at=ends_at,
            offer_cents=offer_cents,
            offer_ends_at=offer_ends_at,
            saw_offer=saw_offer,
        )

        return revenue_activity

