from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_member_row_kind import EarnMemberRowKind
from ..models.earn_member_row_next import EarnMemberRowNext
from ..models.earn_member_row_period import EarnMemberRowPeriod
from ..models.earn_member_row_status import EarnMemberRowStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_member_discount import EarnMemberDiscount
  from ..models.earn_member_source import EarnMemberSource





T = TypeVar("T", bound="EarnMemberRow")



@_attrs_define
class EarnMemberRow:
    """ 
        Attributes:
            access_paused (bool):
            amount_cents (int):
            contact_id (str):
            currency (str):
            email (str):
            erased (bool):
            hidden (bool):
            id (str):
            kind (EarnMemberRowKind):
            legacy (bool):
            ltv_cents (int):
            name (str):
            next_ (EarnMemberRowNext):
            period (EarnMemberRowPeriod):
            period_shown (bool):
            refunded_cents (int):
            since (datetime.datetime):
            source (EarnMemberSource):
            status (EarnMemberRowStatus):
            test (bool):
            tier (str):
            tier_name (str):
            cancel_at (datetime.datetime | Unset):
            cancel_reason (str | Unset):
            discount (EarnMemberDiscount | Unset):
            end_reason (str | Unset):
            gift_from (str | Unset):
            gift_months (int | Unset):
            next_at (datetime.datetime | Unset):
            tier_id (str | Unset):
     """

    access_paused: bool
    amount_cents: int
    contact_id: str
    currency: str
    email: str
    erased: bool
    hidden: bool
    id: str
    kind: EarnMemberRowKind
    legacy: bool
    ltv_cents: int
    name: str
    next_: EarnMemberRowNext
    period: EarnMemberRowPeriod
    period_shown: bool
    refunded_cents: int
    since: datetime.datetime
    source: EarnMemberSource
    status: EarnMemberRowStatus
    test: bool
    tier: str
    tier_name: str
    cancel_at: datetime.datetime | Unset = UNSET
    cancel_reason: str | Unset = UNSET
    discount: EarnMemberDiscount | Unset = UNSET
    end_reason: str | Unset = UNSET
    gift_from: str | Unset = UNSET
    gift_months: int | Unset = UNSET
    next_at: datetime.datetime | Unset = UNSET
    tier_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_member_discount import EarnMemberDiscount # noqa: PLC0415
        from ..models.earn_member_source import EarnMemberSource # noqa: PLC0415
        access_paused = self.access_paused

        amount_cents = self.amount_cents

        contact_id = self.contact_id

        currency = self.currency

        email = self.email

        erased = self.erased

        hidden = self.hidden

        id = self.id

        kind = self.kind.value

        legacy = self.legacy

        ltv_cents = self.ltv_cents

        name = self.name

        next_ = self.next_.value

        period = self.period.value

        period_shown = self.period_shown

        refunded_cents = self.refunded_cents

        since = self.since.isoformat()

        source = self.source.to_dict()

        status = self.status.value

        test = self.test

        tier = self.tier

        tier_name = self.tier_name

        cancel_at: str | Unset = UNSET
        if not isinstance(self.cancel_at, Unset):
            cancel_at = self.cancel_at.isoformat()

        cancel_reason = self.cancel_reason

        discount: dict[str, Any] | Unset = UNSET
        if not isinstance(self.discount, Unset):
            discount = self.discount.to_dict()

        end_reason = self.end_reason

        gift_from = self.gift_from

        gift_months = self.gift_months

        next_at: str | Unset = UNSET
        if not isinstance(self.next_at, Unset):
            next_at = self.next_at.isoformat()

        tier_id = self.tier_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accessPaused": access_paused,
            "amountCents": amount_cents,
            "contactId": contact_id,
            "currency": currency,
            "email": email,
            "erased": erased,
            "hidden": hidden,
            "id": id,
            "kind": kind,
            "legacy": legacy,
            "ltvCents": ltv_cents,
            "name": name,
            "next": next_,
            "period": period,
            "periodShown": period_shown,
            "refundedCents": refunded_cents,
            "since": since,
            "source": source,
            "status": status,
            "test": test,
            "tier": tier,
            "tierName": tier_name,
        })
        if cancel_at is not UNSET:
            field_dict["cancelAt"] = cancel_at
        if cancel_reason is not UNSET:
            field_dict["cancelReason"] = cancel_reason
        if discount is not UNSET:
            field_dict["discount"] = discount
        if end_reason is not UNSET:
            field_dict["endReason"] = end_reason
        if gift_from is not UNSET:
            field_dict["giftFrom"] = gift_from
        if gift_months is not UNSET:
            field_dict["giftMonths"] = gift_months
        if next_at is not UNSET:
            field_dict["nextAt"] = next_at
        if tier_id is not UNSET:
            field_dict["tierId"] = tier_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_member_discount import EarnMemberDiscount # noqa: PLC0415
        from ..models.earn_member_source import EarnMemberSource # noqa: PLC0415
        d = dict(src_dict)
        access_paused = d.pop("accessPaused")

        amount_cents = d.pop("amountCents")

        contact_id = d.pop("contactId")

        currency = d.pop("currency")

        email = d.pop("email")

        erased = d.pop("erased")

        hidden = d.pop("hidden")

        id = d.pop("id")

        kind = EarnMemberRowKind(d.pop("kind"))




        legacy = d.pop("legacy")

        ltv_cents = d.pop("ltvCents")

        name = d.pop("name")

        next_ = EarnMemberRowNext(d.pop("next"))




        period = EarnMemberRowPeriod(d.pop("period"))




        period_shown = d.pop("periodShown")

        refunded_cents = d.pop("refundedCents")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        source = EarnMemberSource.from_dict(d.pop("source"))




        status = EarnMemberRowStatus(d.pop("status"))




        test = d.pop("test")

        tier = d.pop("tier")

        tier_name = d.pop("tierName")

        _cancel_at = d.pop("cancelAt", UNSET)
        cancel_at: datetime.datetime | Unset
        if isinstance(_cancel_at,  Unset):
            cancel_at = UNSET
        else:
            cancel_at = datetime.datetime.fromisoformat(_cancel_at)




        cancel_reason = d.pop("cancelReason", UNSET)

        _discount = d.pop("discount", UNSET)
        discount: EarnMemberDiscount | Unset
        if isinstance(_discount,  Unset):
            discount = UNSET
        else:
            discount = EarnMemberDiscount.from_dict(_discount)




        end_reason = d.pop("endReason", UNSET)

        gift_from = d.pop("giftFrom", UNSET)

        gift_months = d.pop("giftMonths", UNSET)

        _next_at = d.pop("nextAt", UNSET)
        next_at: datetime.datetime | Unset
        if isinstance(_next_at,  Unset):
            next_at = UNSET
        else:
            next_at = datetime.datetime.fromisoformat(_next_at)




        tier_id = d.pop("tierId", UNSET)

        earn_member_row = cls(
            access_paused=access_paused,
            amount_cents=amount_cents,
            contact_id=contact_id,
            currency=currency,
            email=email,
            erased=erased,
            hidden=hidden,
            id=id,
            kind=kind,
            legacy=legacy,
            ltv_cents=ltv_cents,
            name=name,
            next_=next_,
            period=period,
            period_shown=period_shown,
            refunded_cents=refunded_cents,
            since=since,
            source=source,
            status=status,
            test=test,
            tier=tier,
            tier_name=tier_name,
            cancel_at=cancel_at,
            cancel_reason=cancel_reason,
            discount=discount,
            end_reason=end_reason,
            gift_from=gift_from,
            gift_months=gift_months,
            next_at=next_at,
            tier_id=tier_id,
        )

        return earn_member_row

