from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_activity_item_credited_kind import CommerceActivityItemCreditedKind
from ..models.commerce_activity_item_kind import CommerceActivityItemKind
from ..models.commerce_activity_item_matched import CommerceActivityItemMatched
from ..models.commerce_activity_item_tone import CommerceActivityItemTone
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceActivityItem")



@_attrs_define
class CommerceActivityItem:
    """ 
        Attributes:
            accepted (bool):
            amount (int):
            at (datetime.datetime):
            contact (bool):
            flows (list[str]):
            id (str):
            kind (CommerceActivityItemKind):
            lists (list[str]):
            missed (int):
            polled (bool):
            products (int):
            recovered (bool):
            tone (CommerceActivityItemTone):
            by (str | Unset):
            contact_id (str | Unset):
            credited (str | Unset):
            credited_kind (CommerceActivityItemCreditedKind | Unset):
            credited_step (int | Unset):
            currency (str | Unset):
            customer (str | Unset):
            flow (str | Unset):
            hours (int | Unset):
            matched (CommerceActivityItemMatched | Unset):
            message (str | Unset):
            order (str | Unset):
            reason (str | Unset):
            source (str | Unset):
            topic (str | Unset):
            until (datetime.datetime | Unset):
            window (int | Unset):
     """

    accepted: bool
    amount: int
    at: datetime.datetime
    contact: bool
    flows: list[str]
    id: str
    kind: CommerceActivityItemKind
    lists: list[str]
    missed: int
    polled: bool
    products: int
    recovered: bool
    tone: CommerceActivityItemTone
    by: str | Unset = UNSET
    contact_id: str | Unset = UNSET
    credited: str | Unset = UNSET
    credited_kind: CommerceActivityItemCreditedKind | Unset = UNSET
    credited_step: int | Unset = UNSET
    currency: str | Unset = UNSET
    customer: str | Unset = UNSET
    flow: str | Unset = UNSET
    hours: int | Unset = UNSET
    matched: CommerceActivityItemMatched | Unset = UNSET
    message: str | Unset = UNSET
    order: str | Unset = UNSET
    reason: str | Unset = UNSET
    source: str | Unset = UNSET
    topic: str | Unset = UNSET
    until: datetime.datetime | Unset = UNSET
    window: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        accepted = self.accepted

        amount = self.amount

        at = self.at.isoformat()

        contact = self.contact

        flows = self.flows



        id = self.id

        kind = self.kind.value

        lists = self.lists



        missed = self.missed

        polled = self.polled

        products = self.products

        recovered = self.recovered

        tone = self.tone.value

        by = self.by

        contact_id = self.contact_id

        credited = self.credited

        credited_kind: str | Unset = UNSET
        if not isinstance(self.credited_kind, Unset):
            credited_kind = self.credited_kind.value


        credited_step = self.credited_step

        currency = self.currency

        customer = self.customer

        flow = self.flow

        hours = self.hours

        matched: str | Unset = UNSET
        if not isinstance(self.matched, Unset):
            matched = self.matched.value


        message = self.message

        order = self.order

        reason = self.reason

        source = self.source

        topic = self.topic

        until: str | Unset = UNSET
        if not isinstance(self.until, Unset):
            until = self.until.isoformat()

        window = self.window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accepted": accepted,
            "amount": amount,
            "at": at,
            "contact": contact,
            "flows": flows,
            "id": id,
            "kind": kind,
            "lists": lists,
            "missed": missed,
            "polled": polled,
            "products": products,
            "recovered": recovered,
            "tone": tone,
        })
        if by is not UNSET:
            field_dict["by"] = by
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if credited is not UNSET:
            field_dict["credited"] = credited
        if credited_kind is not UNSET:
            field_dict["creditedKind"] = credited_kind
        if credited_step is not UNSET:
            field_dict["creditedStep"] = credited_step
        if currency is not UNSET:
            field_dict["currency"] = currency
        if customer is not UNSET:
            field_dict["customer"] = customer
        if flow is not UNSET:
            field_dict["flow"] = flow
        if hours is not UNSET:
            field_dict["hours"] = hours
        if matched is not UNSET:
            field_dict["matched"] = matched
        if message is not UNSET:
            field_dict["message"] = message
        if order is not UNSET:
            field_dict["order"] = order
        if reason is not UNSET:
            field_dict["reason"] = reason
        if source is not UNSET:
            field_dict["source"] = source
        if topic is not UNSET:
            field_dict["topic"] = topic
        if until is not UNSET:
            field_dict["until"] = until
        if window is not UNSET:
            field_dict["window"] = window

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        accepted = d.pop("accepted")

        amount = d.pop("amount")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        contact = d.pop("contact")

        flows = cast(list[str], d.pop("flows"))


        id = d.pop("id")

        kind = CommerceActivityItemKind(d.pop("kind"))




        lists = cast(list[str], d.pop("lists"))


        missed = d.pop("missed")

        polled = d.pop("polled")

        products = d.pop("products")

        recovered = d.pop("recovered")

        tone = CommerceActivityItemTone(d.pop("tone"))




        by = d.pop("by", UNSET)

        contact_id = d.pop("contactId", UNSET)

        credited = d.pop("credited", UNSET)

        _credited_kind = d.pop("creditedKind", UNSET)
        credited_kind: CommerceActivityItemCreditedKind | Unset
        if isinstance(_credited_kind,  Unset):
            credited_kind = UNSET
        else:
            credited_kind = CommerceActivityItemCreditedKind(_credited_kind)




        credited_step = d.pop("creditedStep", UNSET)

        currency = d.pop("currency", UNSET)

        customer = d.pop("customer", UNSET)

        flow = d.pop("flow", UNSET)

        hours = d.pop("hours", UNSET)

        _matched = d.pop("matched", UNSET)
        matched: CommerceActivityItemMatched | Unset
        if isinstance(_matched,  Unset):
            matched = UNSET
        else:
            matched = CommerceActivityItemMatched(_matched)




        message = d.pop("message", UNSET)

        order = d.pop("order", UNSET)

        reason = d.pop("reason", UNSET)

        source = d.pop("source", UNSET)

        topic = d.pop("topic", UNSET)

        _until = d.pop("until", UNSET)
        until: datetime.datetime | Unset
        if isinstance(_until,  Unset):
            until = UNSET
        else:
            until = datetime.datetime.fromisoformat(_until)




        window = d.pop("window", UNSET)

        commerce_activity_item = cls(
            accepted=accepted,
            amount=amount,
            at=at,
            contact=contact,
            flows=flows,
            id=id,
            kind=kind,
            lists=lists,
            missed=missed,
            polled=polled,
            products=products,
            recovered=recovered,
            tone=tone,
            by=by,
            contact_id=contact_id,
            credited=credited,
            credited_kind=credited_kind,
            credited_step=credited_step,
            currency=currency,
            customer=customer,
            flow=flow,
            hours=hours,
            matched=matched,
            message=message,
            order=order,
            reason=reason,
            source=source,
            topic=topic,
            until=until,
            window=window,
        )

        return commerce_activity_item

