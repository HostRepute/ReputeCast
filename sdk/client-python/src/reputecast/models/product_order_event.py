from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_order_event_kind import ProductOrderEventKind
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProductOrderEvent")



@_attrs_define
class ProductOrderEvent:
    """ 
        Attributes:
            at (datetime.datetime):
            by (str):
            kind (ProductOrderEventKind):
            allowed (int | Unset):
            amount (int | Unset):
            auto (bool | Unset): Done by the system, as the product's setting says
            by_stripe (bool | Unset): Done in Stripe's dashboard
            bytes_ (int | Unset):
            count (int | Unset):
            currency (str | Unset):
            email (str | Unset):
            file (str | Unset):
            note (str | Unset):
            outcome (str | Unset):
            page (str | Unset):
            provider (str | Unset):
            reason (str | Unset):
            respond_by (datetime.datetime | Unset):
            seconds (float | Unset):
            stripe (str | Unset):
            template (str | Unset):
            ticked (bool | Unset):
            used (int | Unset):
            version (int | Unset):
     """

    at: datetime.datetime
    by: str
    kind: ProductOrderEventKind
    allowed: int | Unset = UNSET
    amount: int | Unset = UNSET
    auto: bool | Unset = UNSET
    by_stripe: bool | Unset = UNSET
    bytes_: int | Unset = UNSET
    count: int | Unset = UNSET
    currency: str | Unset = UNSET
    email: str | Unset = UNSET
    file: str | Unset = UNSET
    note: str | Unset = UNSET
    outcome: str | Unset = UNSET
    page: str | Unset = UNSET
    provider: str | Unset = UNSET
    reason: str | Unset = UNSET
    respond_by: datetime.datetime | Unset = UNSET
    seconds: float | Unset = UNSET
    stripe: str | Unset = UNSET
    template: str | Unset = UNSET
    ticked: bool | Unset = UNSET
    used: int | Unset = UNSET
    version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        by = self.by

        kind = self.kind.value

        allowed = self.allowed

        amount = self.amount

        auto = self.auto

        by_stripe = self.by_stripe

        bytes_ = self.bytes_

        count = self.count

        currency = self.currency

        email = self.email

        file = self.file

        note = self.note

        outcome = self.outcome

        page = self.page

        provider = self.provider

        reason = self.reason

        respond_by: str | Unset = UNSET
        if not isinstance(self.respond_by, Unset):
            respond_by = self.respond_by.isoformat()

        seconds = self.seconds

        stripe = self.stripe

        template = self.template

        ticked = self.ticked

        used = self.used

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "by": by,
            "kind": kind,
        })
        if allowed is not UNSET:
            field_dict["allowed"] = allowed
        if amount is not UNSET:
            field_dict["amount"] = amount
        if auto is not UNSET:
            field_dict["auto"] = auto
        if by_stripe is not UNSET:
            field_dict["byStripe"] = by_stripe
        if bytes_ is not UNSET:
            field_dict["bytes"] = bytes_
        if count is not UNSET:
            field_dict["count"] = count
        if currency is not UNSET:
            field_dict["currency"] = currency
        if email is not UNSET:
            field_dict["email"] = email
        if file is not UNSET:
            field_dict["file"] = file
        if note is not UNSET:
            field_dict["note"] = note
        if outcome is not UNSET:
            field_dict["outcome"] = outcome
        if page is not UNSET:
            field_dict["page"] = page
        if provider is not UNSET:
            field_dict["provider"] = provider
        if reason is not UNSET:
            field_dict["reason"] = reason
        if respond_by is not UNSET:
            field_dict["respondBy"] = respond_by
        if seconds is not UNSET:
            field_dict["seconds"] = seconds
        if stripe is not UNSET:
            field_dict["stripe"] = stripe
        if template is not UNSET:
            field_dict["template"] = template
        if ticked is not UNSET:
            field_dict["ticked"] = ticked
        if used is not UNSET:
            field_dict["used"] = used
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        by = d.pop("by")

        kind = ProductOrderEventKind(d.pop("kind"))




        allowed = d.pop("allowed", UNSET)

        amount = d.pop("amount", UNSET)

        auto = d.pop("auto", UNSET)

        by_stripe = d.pop("byStripe", UNSET)

        bytes_ = d.pop("bytes", UNSET)

        count = d.pop("count", UNSET)

        currency = d.pop("currency", UNSET)

        email = d.pop("email", UNSET)

        file = d.pop("file", UNSET)

        note = d.pop("note", UNSET)

        outcome = d.pop("outcome", UNSET)

        page = d.pop("page", UNSET)

        provider = d.pop("provider", UNSET)

        reason = d.pop("reason", UNSET)

        _respond_by = d.pop("respondBy", UNSET)
        respond_by: datetime.datetime | Unset
        if isinstance(_respond_by,  Unset):
            respond_by = UNSET
        else:
            respond_by = datetime.datetime.fromisoformat(_respond_by)




        seconds = d.pop("seconds", UNSET)

        stripe = d.pop("stripe", UNSET)

        template = d.pop("template", UNSET)

        ticked = d.pop("ticked", UNSET)

        used = d.pop("used", UNSET)

        version = d.pop("version", UNSET)

        product_order_event = cls(
            at=at,
            by=by,
            kind=kind,
            allowed=allowed,
            amount=amount,
            auto=auto,
            by_stripe=by_stripe,
            bytes_=bytes_,
            count=count,
            currency=currency,
            email=email,
            file=file,
            note=note,
            outcome=outcome,
            page=page,
            provider=provider,
            reason=reason,
            respond_by=respond_by,
            seconds=seconds,
            stripe=stripe,
            template=template,
            ticked=ticked,
            used=used,
            version=version,
        )

        return product_order_event

