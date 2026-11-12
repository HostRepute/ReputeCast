from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_member_kind import EarnMemberKind
from ..models.earn_member_next import EarnMemberNext
from ..models.earn_member_period import EarnMemberPeriod
from ..models.earn_member_status import EarnMemberStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.earn_member_discount import EarnMemberDiscount
  from ..models.earn_member_source import EarnMemberSource
  from ..models.earn_sync_warning import EarnSyncWarning
  from ..models.earn_timeline_item import EarnTimelineItem





T = TypeVar("T", bound="EarnMember")



@_attrs_define
class EarnMember:
    """ 
        Attributes:
            access_paused (bool):
            amount_cents (int):
            billing_differs (bool):
            can_manage (bool):
            contact_id (str):
            currency (str):
            email (str):
            emails (list[str]):
            erased (bool):
            helpers (list[str]):
            hidden (bool):
            id (str):
            kind (EarnMemberKind):
            legacy (bool):
            link (str):
            lists (list[str]):
            ltv_cents (int):
            name (str):
            next_ (EarnMemberNext):
            paid_cents (int):
            period (EarnMemberPeriod):
            period_shown (bool):
            refunded_cents (int):
            since (datetime.datetime):
            source (EarnMemberSource):
            status (EarnMemberStatus):
            tax_added (bool):
            test (bool):
            tier (str):
            tier_name (str):
            timeline (list[EarnTimelineItem]):
            billing_email (str | Unset):
            cancel_at (datetime.datetime | Unset):
            cancel_reason (str | Unset):
            canceled_at (datetime.datetime | Unset):
            canceled_by (str | Unset):
            clicked_at (datetime.datetime | Unset):
            customer_id (str | Unset):
            customer_url (str | Unset):
            discount (EarnMemberDiscount | Unset):
            end_reason (str | Unset):
            failed_at (datetime.datetime | Unset):
            failure_reason (str | Unset):
            failure_text (str | Unset):
            gift_from (str | Unset):
            gift_months (int | Unset):
            grant_note (str | Unset):
            grant_until (datetime.datetime | Unset):
            imported_at (datetime.datetime | Unset):
            matched_by (str | Unset):
            next_at (datetime.datetime | Unset):
            renewal_reminder (datetime.datetime | Unset):
            subscription_id (str | Unset):
            subscription_url (str | Unset):
            tier_id (str | Unset):
            warning (EarnSyncWarning | Unset):
     """

    access_paused: bool
    amount_cents: int
    billing_differs: bool
    can_manage: bool
    contact_id: str
    currency: str
    email: str
    emails: list[str]
    erased: bool
    helpers: list[str]
    hidden: bool
    id: str
    kind: EarnMemberKind
    legacy: bool
    link: str
    lists: list[str]
    ltv_cents: int
    name: str
    next_: EarnMemberNext
    paid_cents: int
    period: EarnMemberPeriod
    period_shown: bool
    refunded_cents: int
    since: datetime.datetime
    source: EarnMemberSource
    status: EarnMemberStatus
    tax_added: bool
    test: bool
    tier: str
    tier_name: str
    timeline: list[EarnTimelineItem]
    billing_email: str | Unset = UNSET
    cancel_at: datetime.datetime | Unset = UNSET
    cancel_reason: str | Unset = UNSET
    canceled_at: datetime.datetime | Unset = UNSET
    canceled_by: str | Unset = UNSET
    clicked_at: datetime.datetime | Unset = UNSET
    customer_id: str | Unset = UNSET
    customer_url: str | Unset = UNSET
    discount: EarnMemberDiscount | Unset = UNSET
    end_reason: str | Unset = UNSET
    failed_at: datetime.datetime | Unset = UNSET
    failure_reason: str | Unset = UNSET
    failure_text: str | Unset = UNSET
    gift_from: str | Unset = UNSET
    gift_months: int | Unset = UNSET
    grant_note: str | Unset = UNSET
    grant_until: datetime.datetime | Unset = UNSET
    imported_at: datetime.datetime | Unset = UNSET
    matched_by: str | Unset = UNSET
    next_at: datetime.datetime | Unset = UNSET
    renewal_reminder: datetime.datetime | Unset = UNSET
    subscription_id: str | Unset = UNSET
    subscription_url: str | Unset = UNSET
    tier_id: str | Unset = UNSET
    warning: EarnSyncWarning | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_member_discount import EarnMemberDiscount # noqa: PLC0415
        from ..models.earn_member_source import EarnMemberSource # noqa: PLC0415
        from ..models.earn_sync_warning import EarnSyncWarning # noqa: PLC0415
        from ..models.earn_timeline_item import EarnTimelineItem # noqa: PLC0415
        access_paused = self.access_paused

        amount_cents = self.amount_cents

        billing_differs = self.billing_differs

        can_manage = self.can_manage

        contact_id = self.contact_id

        currency = self.currency

        email = self.email

        emails = self.emails



        erased = self.erased

        helpers = self.helpers



        hidden = self.hidden

        id = self.id

        kind = self.kind.value

        legacy = self.legacy

        link = self.link

        lists = self.lists



        ltv_cents = self.ltv_cents

        name = self.name

        next_ = self.next_.value

        paid_cents = self.paid_cents

        period = self.period.value

        period_shown = self.period_shown

        refunded_cents = self.refunded_cents

        since = self.since.isoformat()

        source = self.source.to_dict()

        status = self.status.value

        tax_added = self.tax_added

        test = self.test

        tier = self.tier

        tier_name = self.tier_name

        timeline = []
        for timeline_item_data in self.timeline:
            timeline_item = timeline_item_data.to_dict()
            timeline.append(timeline_item)



        billing_email = self.billing_email

        cancel_at: str | Unset = UNSET
        if not isinstance(self.cancel_at, Unset):
            cancel_at = self.cancel_at.isoformat()

        cancel_reason = self.cancel_reason

        canceled_at: str | Unset = UNSET
        if not isinstance(self.canceled_at, Unset):
            canceled_at = self.canceled_at.isoformat()

        canceled_by = self.canceled_by

        clicked_at: str | Unset = UNSET
        if not isinstance(self.clicked_at, Unset):
            clicked_at = self.clicked_at.isoformat()

        customer_id = self.customer_id

        customer_url = self.customer_url

        discount: dict[str, Any] | Unset = UNSET
        if not isinstance(self.discount, Unset):
            discount = self.discount.to_dict()

        end_reason = self.end_reason

        failed_at: str | Unset = UNSET
        if not isinstance(self.failed_at, Unset):
            failed_at = self.failed_at.isoformat()

        failure_reason = self.failure_reason

        failure_text = self.failure_text

        gift_from = self.gift_from

        gift_months = self.gift_months

        grant_note = self.grant_note

        grant_until: str | Unset = UNSET
        if not isinstance(self.grant_until, Unset):
            grant_until = self.grant_until.isoformat()

        imported_at: str | Unset = UNSET
        if not isinstance(self.imported_at, Unset):
            imported_at = self.imported_at.isoformat()

        matched_by = self.matched_by

        next_at: str | Unset = UNSET
        if not isinstance(self.next_at, Unset):
            next_at = self.next_at.isoformat()

        renewal_reminder: str | Unset = UNSET
        if not isinstance(self.renewal_reminder, Unset):
            renewal_reminder = self.renewal_reminder.isoformat()

        subscription_id = self.subscription_id

        subscription_url = self.subscription_url

        tier_id = self.tier_id

        warning: dict[str, Any] | Unset = UNSET
        if not isinstance(self.warning, Unset):
            warning = self.warning.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "accessPaused": access_paused,
            "amountCents": amount_cents,
            "billingDiffers": billing_differs,
            "canManage": can_manage,
            "contactId": contact_id,
            "currency": currency,
            "email": email,
            "emails": emails,
            "erased": erased,
            "helpers": helpers,
            "hidden": hidden,
            "id": id,
            "kind": kind,
            "legacy": legacy,
            "link": link,
            "lists": lists,
            "ltvCents": ltv_cents,
            "name": name,
            "next": next_,
            "paidCents": paid_cents,
            "period": period,
            "periodShown": period_shown,
            "refundedCents": refunded_cents,
            "since": since,
            "source": source,
            "status": status,
            "taxAdded": tax_added,
            "test": test,
            "tier": tier,
            "tierName": tier_name,
            "timeline": timeline,
        })
        if billing_email is not UNSET:
            field_dict["billingEmail"] = billing_email
        if cancel_at is not UNSET:
            field_dict["cancelAt"] = cancel_at
        if cancel_reason is not UNSET:
            field_dict["cancelReason"] = cancel_reason
        if canceled_at is not UNSET:
            field_dict["canceledAt"] = canceled_at
        if canceled_by is not UNSET:
            field_dict["canceledBy"] = canceled_by
        if clicked_at is not UNSET:
            field_dict["clickedAt"] = clicked_at
        if customer_id is not UNSET:
            field_dict["customerId"] = customer_id
        if customer_url is not UNSET:
            field_dict["customerUrl"] = customer_url
        if discount is not UNSET:
            field_dict["discount"] = discount
        if end_reason is not UNSET:
            field_dict["endReason"] = end_reason
        if failed_at is not UNSET:
            field_dict["failedAt"] = failed_at
        if failure_reason is not UNSET:
            field_dict["failureReason"] = failure_reason
        if failure_text is not UNSET:
            field_dict["failureText"] = failure_text
        if gift_from is not UNSET:
            field_dict["giftFrom"] = gift_from
        if gift_months is not UNSET:
            field_dict["giftMonths"] = gift_months
        if grant_note is not UNSET:
            field_dict["grantNote"] = grant_note
        if grant_until is not UNSET:
            field_dict["grantUntil"] = grant_until
        if imported_at is not UNSET:
            field_dict["importedAt"] = imported_at
        if matched_by is not UNSET:
            field_dict["matchedBy"] = matched_by
        if next_at is not UNSET:
            field_dict["nextAt"] = next_at
        if renewal_reminder is not UNSET:
            field_dict["renewalReminder"] = renewal_reminder
        if subscription_id is not UNSET:
            field_dict["subscriptionId"] = subscription_id
        if subscription_url is not UNSET:
            field_dict["subscriptionUrl"] = subscription_url
        if tier_id is not UNSET:
            field_dict["tierId"] = tier_id
        if warning is not UNSET:
            field_dict["warning"] = warning

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_member_discount import EarnMemberDiscount # noqa: PLC0415
        from ..models.earn_member_source import EarnMemberSource # noqa: PLC0415
        from ..models.earn_sync_warning import EarnSyncWarning # noqa: PLC0415
        from ..models.earn_timeline_item import EarnTimelineItem # noqa: PLC0415
        d = dict(src_dict)
        access_paused = d.pop("accessPaused")

        amount_cents = d.pop("amountCents")

        billing_differs = d.pop("billingDiffers")

        can_manage = d.pop("canManage")

        contact_id = d.pop("contactId")

        currency = d.pop("currency")

        email = d.pop("email")

        emails = cast(list[str], d.pop("emails"))


        erased = d.pop("erased")

        helpers = cast(list[str], d.pop("helpers"))


        hidden = d.pop("hidden")

        id = d.pop("id")

        kind = EarnMemberKind(d.pop("kind"))




        legacy = d.pop("legacy")

        link = d.pop("link")

        lists = cast(list[str], d.pop("lists"))


        ltv_cents = d.pop("ltvCents")

        name = d.pop("name")

        next_ = EarnMemberNext(d.pop("next"))




        paid_cents = d.pop("paidCents")

        period = EarnMemberPeriod(d.pop("period"))




        period_shown = d.pop("periodShown")

        refunded_cents = d.pop("refundedCents")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        source = EarnMemberSource.from_dict(d.pop("source"))




        status = EarnMemberStatus(d.pop("status"))




        tax_added = d.pop("taxAdded")

        test = d.pop("test")

        tier = d.pop("tier")

        tier_name = d.pop("tierName")

        timeline = []
        _timeline = d.pop("timeline")
        for timeline_item_data in (_timeline):
            timeline_item = EarnTimelineItem.from_dict(timeline_item_data)



            timeline.append(timeline_item)


        billing_email = d.pop("billingEmail", UNSET)

        _cancel_at = d.pop("cancelAt", UNSET)
        cancel_at: datetime.datetime | Unset
        if isinstance(_cancel_at,  Unset):
            cancel_at = UNSET
        else:
            cancel_at = datetime.datetime.fromisoformat(_cancel_at)




        cancel_reason = d.pop("cancelReason", UNSET)

        _canceled_at = d.pop("canceledAt", UNSET)
        canceled_at: datetime.datetime | Unset
        if isinstance(_canceled_at,  Unset):
            canceled_at = UNSET
        else:
            canceled_at = datetime.datetime.fromisoformat(_canceled_at)




        canceled_by = d.pop("canceledBy", UNSET)

        _clicked_at = d.pop("clickedAt", UNSET)
        clicked_at: datetime.datetime | Unset
        if isinstance(_clicked_at,  Unset):
            clicked_at = UNSET
        else:
            clicked_at = datetime.datetime.fromisoformat(_clicked_at)




        customer_id = d.pop("customerId", UNSET)

        customer_url = d.pop("customerUrl", UNSET)

        _discount = d.pop("discount", UNSET)
        discount: EarnMemberDiscount | Unset
        if isinstance(_discount,  Unset):
            discount = UNSET
        else:
            discount = EarnMemberDiscount.from_dict(_discount)




        end_reason = d.pop("endReason", UNSET)

        _failed_at = d.pop("failedAt", UNSET)
        failed_at: datetime.datetime | Unset
        if isinstance(_failed_at,  Unset):
            failed_at = UNSET
        else:
            failed_at = datetime.datetime.fromisoformat(_failed_at)




        failure_reason = d.pop("failureReason", UNSET)

        failure_text = d.pop("failureText", UNSET)

        gift_from = d.pop("giftFrom", UNSET)

        gift_months = d.pop("giftMonths", UNSET)

        grant_note = d.pop("grantNote", UNSET)

        _grant_until = d.pop("grantUntil", UNSET)
        grant_until: datetime.datetime | Unset
        if isinstance(_grant_until,  Unset):
            grant_until = UNSET
        else:
            grant_until = datetime.datetime.fromisoformat(_grant_until)




        _imported_at = d.pop("importedAt", UNSET)
        imported_at: datetime.datetime | Unset
        if isinstance(_imported_at,  Unset):
            imported_at = UNSET
        else:
            imported_at = datetime.datetime.fromisoformat(_imported_at)




        matched_by = d.pop("matchedBy", UNSET)

        _next_at = d.pop("nextAt", UNSET)
        next_at: datetime.datetime | Unset
        if isinstance(_next_at,  Unset):
            next_at = UNSET
        else:
            next_at = datetime.datetime.fromisoformat(_next_at)




        _renewal_reminder = d.pop("renewalReminder", UNSET)
        renewal_reminder: datetime.datetime | Unset
        if isinstance(_renewal_reminder,  Unset):
            renewal_reminder = UNSET
        else:
            renewal_reminder = datetime.datetime.fromisoformat(_renewal_reminder)




        subscription_id = d.pop("subscriptionId", UNSET)

        subscription_url = d.pop("subscriptionUrl", UNSET)

        tier_id = d.pop("tierId", UNSET)

        _warning = d.pop("warning", UNSET)
        warning: EarnSyncWarning | Unset
        if isinstance(_warning,  Unset):
            warning = UNSET
        else:
            warning = EarnSyncWarning.from_dict(_warning)




        earn_member = cls(
            access_paused=access_paused,
            amount_cents=amount_cents,
            billing_differs=billing_differs,
            can_manage=can_manage,
            contact_id=contact_id,
            currency=currency,
            email=email,
            emails=emails,
            erased=erased,
            helpers=helpers,
            hidden=hidden,
            id=id,
            kind=kind,
            legacy=legacy,
            link=link,
            lists=lists,
            ltv_cents=ltv_cents,
            name=name,
            next_=next_,
            paid_cents=paid_cents,
            period=period,
            period_shown=period_shown,
            refunded_cents=refunded_cents,
            since=since,
            source=source,
            status=status,
            tax_added=tax_added,
            test=test,
            tier=tier,
            tier_name=tier_name,
            timeline=timeline,
            billing_email=billing_email,
            cancel_at=cancel_at,
            cancel_reason=cancel_reason,
            canceled_at=canceled_at,
            canceled_by=canceled_by,
            clicked_at=clicked_at,
            customer_id=customer_id,
            customer_url=customer_url,
            discount=discount,
            end_reason=end_reason,
            failed_at=failed_at,
            failure_reason=failure_reason,
            failure_text=failure_text,
            gift_from=gift_from,
            gift_months=gift_months,
            grant_note=grant_note,
            grant_until=grant_until,
            imported_at=imported_at,
            matched_by=matched_by,
            next_at=next_at,
            renewal_reminder=renewal_reminder,
            subscription_id=subscription_id,
            subscription_url=subscription_url,
            tier_id=tier_id,
            warning=warning,
        )

        return earn_member

