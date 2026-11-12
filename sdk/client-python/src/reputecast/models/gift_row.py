from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.gift_row_status import GiftRowStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="GiftRow")



@_attrs_define
class GiftRow:
    """ 
        Attributes:
            amount_cents (int):
            bought_at (datetime.datetime):
            buyer (str):
            buyer_email (str):
            currency (str):
            email_status (str):
            id (str):
            in_person (bool):
            months (int):
            recipient_email (str):
            recipient_name (str):
            short (str):
            status (GiftRowStatus):
            tier (str):
            timezone (str):
            access_ends (datetime.datetime | Unset):
            deliver_at (datetime.datetime | Unset):
            delivered_at (datetime.datetime | Unset):
            link_expires_at (datetime.datetime | Unset):
            payment_url (str | Unset):
            redeemed_at (datetime.datetime | Unset):
            redeemed_by (str | Unset):
            refunded_at (datetime.datetime | Unset):
            remind_at (datetime.datetime | Unset):
            reminded_at (datetime.datetime | Unset):
     """

    amount_cents: int
    bought_at: datetime.datetime
    buyer: str
    buyer_email: str
    currency: str
    email_status: str
    id: str
    in_person: bool
    months: int
    recipient_email: str
    recipient_name: str
    short: str
    status: GiftRowStatus
    tier: str
    timezone: str
    access_ends: datetime.datetime | Unset = UNSET
    deliver_at: datetime.datetime | Unset = UNSET
    delivered_at: datetime.datetime | Unset = UNSET
    link_expires_at: datetime.datetime | Unset = UNSET
    payment_url: str | Unset = UNSET
    redeemed_at: datetime.datetime | Unset = UNSET
    redeemed_by: str | Unset = UNSET
    refunded_at: datetime.datetime | Unset = UNSET
    remind_at: datetime.datetime | Unset = UNSET
    reminded_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        amount_cents = self.amount_cents

        bought_at = self.bought_at.isoformat()

        buyer = self.buyer

        buyer_email = self.buyer_email

        currency = self.currency

        email_status = self.email_status

        id = self.id

        in_person = self.in_person

        months = self.months

        recipient_email = self.recipient_email

        recipient_name = self.recipient_name

        short = self.short

        status = self.status.value

        tier = self.tier

        timezone = self.timezone

        access_ends: str | Unset = UNSET
        if not isinstance(self.access_ends, Unset):
            access_ends = self.access_ends.isoformat()

        deliver_at: str | Unset = UNSET
        if not isinstance(self.deliver_at, Unset):
            deliver_at = self.deliver_at.isoformat()

        delivered_at: str | Unset = UNSET
        if not isinstance(self.delivered_at, Unset):
            delivered_at = self.delivered_at.isoformat()

        link_expires_at: str | Unset = UNSET
        if not isinstance(self.link_expires_at, Unset):
            link_expires_at = self.link_expires_at.isoformat()

        payment_url = self.payment_url

        redeemed_at: str | Unset = UNSET
        if not isinstance(self.redeemed_at, Unset):
            redeemed_at = self.redeemed_at.isoformat()

        redeemed_by = self.redeemed_by

        refunded_at: str | Unset = UNSET
        if not isinstance(self.refunded_at, Unset):
            refunded_at = self.refunded_at.isoformat()

        remind_at: str | Unset = UNSET
        if not isinstance(self.remind_at, Unset):
            remind_at = self.remind_at.isoformat()

        reminded_at: str | Unset = UNSET
        if not isinstance(self.reminded_at, Unset):
            reminded_at = self.reminded_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "amountCents": amount_cents,
            "boughtAt": bought_at,
            "buyer": buyer,
            "buyerEmail": buyer_email,
            "currency": currency,
            "emailStatus": email_status,
            "id": id,
            "inPerson": in_person,
            "months": months,
            "recipientEmail": recipient_email,
            "recipientName": recipient_name,
            "short": short,
            "status": status,
            "tier": tier,
            "timezone": timezone,
        })
        if access_ends is not UNSET:
            field_dict["accessEnds"] = access_ends
        if deliver_at is not UNSET:
            field_dict["deliverAt"] = deliver_at
        if delivered_at is not UNSET:
            field_dict["deliveredAt"] = delivered_at
        if link_expires_at is not UNSET:
            field_dict["linkExpiresAt"] = link_expires_at
        if payment_url is not UNSET:
            field_dict["paymentUrl"] = payment_url
        if redeemed_at is not UNSET:
            field_dict["redeemedAt"] = redeemed_at
        if redeemed_by is not UNSET:
            field_dict["redeemedBy"] = redeemed_by
        if refunded_at is not UNSET:
            field_dict["refundedAt"] = refunded_at
        if remind_at is not UNSET:
            field_dict["remindAt"] = remind_at
        if reminded_at is not UNSET:
            field_dict["remindedAt"] = reminded_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        amount_cents = d.pop("amountCents")

        bought_at = datetime.datetime.fromisoformat(d.pop("boughtAt"))




        buyer = d.pop("buyer")

        buyer_email = d.pop("buyerEmail")

        currency = d.pop("currency")

        email_status = d.pop("emailStatus")

        id = d.pop("id")

        in_person = d.pop("inPerson")

        months = d.pop("months")

        recipient_email = d.pop("recipientEmail")

        recipient_name = d.pop("recipientName")

        short = d.pop("short")

        status = GiftRowStatus(d.pop("status"))




        tier = d.pop("tier")

        timezone = d.pop("timezone")

        _access_ends = d.pop("accessEnds", UNSET)
        access_ends: datetime.datetime | Unset
        if isinstance(_access_ends,  Unset):
            access_ends = UNSET
        else:
            access_ends = datetime.datetime.fromisoformat(_access_ends)




        _deliver_at = d.pop("deliverAt", UNSET)
        deliver_at: datetime.datetime | Unset
        if isinstance(_deliver_at,  Unset):
            deliver_at = UNSET
        else:
            deliver_at = datetime.datetime.fromisoformat(_deliver_at)




        _delivered_at = d.pop("deliveredAt", UNSET)
        delivered_at: datetime.datetime | Unset
        if isinstance(_delivered_at,  Unset):
            delivered_at = UNSET
        else:
            delivered_at = datetime.datetime.fromisoformat(_delivered_at)




        _link_expires_at = d.pop("linkExpiresAt", UNSET)
        link_expires_at: datetime.datetime | Unset
        if isinstance(_link_expires_at,  Unset):
            link_expires_at = UNSET
        else:
            link_expires_at = datetime.datetime.fromisoformat(_link_expires_at)




        payment_url = d.pop("paymentUrl", UNSET)

        _redeemed_at = d.pop("redeemedAt", UNSET)
        redeemed_at: datetime.datetime | Unset
        if isinstance(_redeemed_at,  Unset):
            redeemed_at = UNSET
        else:
            redeemed_at = datetime.datetime.fromisoformat(_redeemed_at)




        redeemed_by = d.pop("redeemedBy", UNSET)

        _refunded_at = d.pop("refundedAt", UNSET)
        refunded_at: datetime.datetime | Unset
        if isinstance(_refunded_at,  Unset):
            refunded_at = UNSET
        else:
            refunded_at = datetime.datetime.fromisoformat(_refunded_at)




        _remind_at = d.pop("remindAt", UNSET)
        remind_at: datetime.datetime | Unset
        if isinstance(_remind_at,  Unset):
            remind_at = UNSET
        else:
            remind_at = datetime.datetime.fromisoformat(_remind_at)




        _reminded_at = d.pop("remindedAt", UNSET)
        reminded_at: datetime.datetime | Unset
        if isinstance(_reminded_at,  Unset):
            reminded_at = UNSET
        else:
            reminded_at = datetime.datetime.fromisoformat(_reminded_at)




        gift_row = cls(
            amount_cents=amount_cents,
            bought_at=bought_at,
            buyer=buyer,
            buyer_email=buyer_email,
            currency=currency,
            email_status=email_status,
            id=id,
            in_person=in_person,
            months=months,
            recipient_email=recipient_email,
            recipient_name=recipient_name,
            short=short,
            status=status,
            tier=tier,
            timezone=timezone,
            access_ends=access_ends,
            deliver_at=deliver_at,
            delivered_at=delivered_at,
            link_expires_at=link_expires_at,
            payment_url=payment_url,
            redeemed_at=redeemed_at,
            redeemed_by=redeemed_by,
            refunded_at=refunded_at,
            remind_at=remind_at,
            reminded_at=reminded_at,
        )

        return gift_row

