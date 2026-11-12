from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_timeline_item_data import EarnTimelineItemData





T = TypeVar("T", bound="EarnTimelineItem")



@_attrs_define
class EarnTimelineItem:
    """ 
        Attributes:
            at (datetime.datetime):
            data (EarnTimelineItemData):
            id (str):
            kind (str):
            test (bool):
            actor (str | Unset):
            amount_cents (int | Unset):
            currency (str | Unset):
            message_id (str | Unset):
            message_status (str | Unset):
            provider (str | Unset):
            stripe_event (str | Unset):
     """

    at: datetime.datetime
    data: EarnTimelineItemData
    id: str
    kind: str
    test: bool
    actor: str | Unset = UNSET
    amount_cents: int | Unset = UNSET
    currency: str | Unset = UNSET
    message_id: str | Unset = UNSET
    message_status: str | Unset = UNSET
    provider: str | Unset = UNSET
    stripe_event: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_timeline_item_data import EarnTimelineItemData # noqa: PLC0415
        at = self.at.isoformat()

        data = self.data.to_dict()

        id = self.id

        kind = self.kind

        test = self.test

        actor = self.actor

        amount_cents = self.amount_cents

        currency = self.currency

        message_id = self.message_id

        message_status = self.message_status

        provider = self.provider

        stripe_event = self.stripe_event


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "data": data,
            "id": id,
            "kind": kind,
            "test": test,
        })
        if actor is not UNSET:
            field_dict["actor"] = actor
        if amount_cents is not UNSET:
            field_dict["amountCents"] = amount_cents
        if currency is not UNSET:
            field_dict["currency"] = currency
        if message_id is not UNSET:
            field_dict["messageId"] = message_id
        if message_status is not UNSET:
            field_dict["messageStatus"] = message_status
        if provider is not UNSET:
            field_dict["provider"] = provider
        if stripe_event is not UNSET:
            field_dict["stripeEvent"] = stripe_event

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_timeline_item_data import EarnTimelineItemData # noqa: PLC0415
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        data = EarnTimelineItemData.from_dict(d.pop("data"))




        id = d.pop("id")

        kind = d.pop("kind")

        test = d.pop("test")

        actor = d.pop("actor", UNSET)

        amount_cents = d.pop("amountCents", UNSET)

        currency = d.pop("currency", UNSET)

        message_id = d.pop("messageId", UNSET)

        message_status = d.pop("messageStatus", UNSET)

        provider = d.pop("provider", UNSET)

        stripe_event = d.pop("stripeEvent", UNSET)

        earn_timeline_item = cls(
            at=at,
            data=data,
            id=id,
            kind=kind,
            test=test,
            actor=actor,
            amount_cents=amount_cents,
            currency=currency,
            message_id=message_id,
            message_status=message_status,
            provider=provider,
            stripe_event=stripe_event,
        )

        return earn_timeline_item

