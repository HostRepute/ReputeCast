from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sponsor_request_item_issue_state import SponsorRequestItemIssueState
from ..models.sponsor_request_item_paid_source import SponsorRequestItemPaidSource
from ..models.sponsor_request_item_status import SponsorRequestItemStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.sponsor_ref import SponsorRef





T = TypeVar("T", bound="SponsorRequestItem")



@_attrs_define
class SponsorRequestItem:
    """ 
        Attributes:
            company (str):
            currency (str):
            email (str):
            id (str):
            issue (int):
            issue_on (datetime.datetime):
            issue_state (SponsorRequestItemIssueState):
            list_name (str):
            name (str):
            price_cents (int):
            slot_deleted (bool):
            slot_id (str):
            slot_name (str):
            status (SponsorRequestItemStatus):
            submitted_at (datetime.datetime):
            closed_at (datetime.datetime | Unset):
            declined_reason (str | Unset):
            held_by (str | Unset):
            held_until (datetime.datetime | Unset):
            paid_on (datetime.datetime | Unset):
            paid_source (SponsorRequestItemPaidSource | Unset):
            payment_due_at (datetime.datetime | Unset):
            sponsor (SponsorRef | Unset):
     """

    company: str
    currency: str
    email: str
    id: str
    issue: int
    issue_on: datetime.datetime
    issue_state: SponsorRequestItemIssueState
    list_name: str
    name: str
    price_cents: int
    slot_deleted: bool
    slot_id: str
    slot_name: str
    status: SponsorRequestItemStatus
    submitted_at: datetime.datetime
    closed_at: datetime.datetime | Unset = UNSET
    declined_reason: str | Unset = UNSET
    held_by: str | Unset = UNSET
    held_until: datetime.datetime | Unset = UNSET
    paid_on: datetime.datetime | Unset = UNSET
    paid_source: SponsorRequestItemPaidSource | Unset = UNSET
    payment_due_at: datetime.datetime | Unset = UNSET
    sponsor: SponsorRef | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        company = self.company

        currency = self.currency

        email = self.email

        id = self.id

        issue = self.issue

        issue_on = self.issue_on.isoformat()

        issue_state = self.issue_state.value

        list_name = self.list_name

        name = self.name

        price_cents = self.price_cents

        slot_deleted = self.slot_deleted

        slot_id = self.slot_id

        slot_name = self.slot_name

        status = self.status.value

        submitted_at = self.submitted_at.isoformat()

        closed_at: str | Unset = UNSET
        if not isinstance(self.closed_at, Unset):
            closed_at = self.closed_at.isoformat()

        declined_reason = self.declined_reason

        held_by = self.held_by

        held_until: str | Unset = UNSET
        if not isinstance(self.held_until, Unset):
            held_until = self.held_until.isoformat()

        paid_on: str | Unset = UNSET
        if not isinstance(self.paid_on, Unset):
            paid_on = self.paid_on.isoformat()

        paid_source: str | Unset = UNSET
        if not isinstance(self.paid_source, Unset):
            paid_source = self.paid_source.value


        payment_due_at: str | Unset = UNSET
        if not isinstance(self.payment_due_at, Unset):
            payment_due_at = self.payment_due_at.isoformat()

        sponsor: dict[str, Any] | Unset = UNSET
        if not isinstance(self.sponsor, Unset):
            sponsor = self.sponsor.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "company": company,
            "currency": currency,
            "email": email,
            "id": id,
            "issue": issue,
            "issueOn": issue_on,
            "issueState": issue_state,
            "listName": list_name,
            "name": name,
            "priceCents": price_cents,
            "slotDeleted": slot_deleted,
            "slotId": slot_id,
            "slotName": slot_name,
            "status": status,
            "submittedAt": submitted_at,
        })
        if closed_at is not UNSET:
            field_dict["closedAt"] = closed_at
        if declined_reason is not UNSET:
            field_dict["declinedReason"] = declined_reason
        if held_by is not UNSET:
            field_dict["heldBy"] = held_by
        if held_until is not UNSET:
            field_dict["heldUntil"] = held_until
        if paid_on is not UNSET:
            field_dict["paidOn"] = paid_on
        if paid_source is not UNSET:
            field_dict["paidSource"] = paid_source
        if payment_due_at is not UNSET:
            field_dict["paymentDueAt"] = payment_due_at
        if sponsor is not UNSET:
            field_dict["sponsor"] = sponsor

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sponsor_ref import SponsorRef # noqa: PLC0415
        d = dict(src_dict)
        company = d.pop("company")

        currency = d.pop("currency")

        email = d.pop("email")

        id = d.pop("id")

        issue = d.pop("issue")

        issue_on = datetime.datetime.fromisoformat(d.pop("issueOn"))




        issue_state = SponsorRequestItemIssueState(d.pop("issueState"))




        list_name = d.pop("listName")

        name = d.pop("name")

        price_cents = d.pop("priceCents")

        slot_deleted = d.pop("slotDeleted")

        slot_id = d.pop("slotId")

        slot_name = d.pop("slotName")

        status = SponsorRequestItemStatus(d.pop("status"))




        submitted_at = datetime.datetime.fromisoformat(d.pop("submittedAt"))




        _closed_at = d.pop("closedAt", UNSET)
        closed_at: datetime.datetime | Unset
        if isinstance(_closed_at,  Unset):
            closed_at = UNSET
        else:
            closed_at = datetime.datetime.fromisoformat(_closed_at)




        declined_reason = d.pop("declinedReason", UNSET)

        held_by = d.pop("heldBy", UNSET)

        _held_until = d.pop("heldUntil", UNSET)
        held_until: datetime.datetime | Unset
        if isinstance(_held_until,  Unset):
            held_until = UNSET
        else:
            held_until = datetime.datetime.fromisoformat(_held_until)




        _paid_on = d.pop("paidOn", UNSET)
        paid_on: datetime.datetime | Unset
        if isinstance(_paid_on,  Unset):
            paid_on = UNSET
        else:
            paid_on = datetime.datetime.fromisoformat(_paid_on)




        _paid_source = d.pop("paidSource", UNSET)
        paid_source: SponsorRequestItemPaidSource | Unset
        if isinstance(_paid_source,  Unset):
            paid_source = UNSET
        else:
            paid_source = SponsorRequestItemPaidSource(_paid_source)




        _payment_due_at = d.pop("paymentDueAt", UNSET)
        payment_due_at: datetime.datetime | Unset
        if isinstance(_payment_due_at,  Unset):
            payment_due_at = UNSET
        else:
            payment_due_at = datetime.datetime.fromisoformat(_payment_due_at)




        _sponsor = d.pop("sponsor", UNSET)
        sponsor: SponsorRef | Unset
        if isinstance(_sponsor,  Unset):
            sponsor = UNSET
        else:
            sponsor = SponsorRef.from_dict(_sponsor)




        sponsor_request_item = cls(
            company=company,
            currency=currency,
            email=email,
            id=id,
            issue=issue,
            issue_on=issue_on,
            issue_state=issue_state,
            list_name=list_name,
            name=name,
            price_cents=price_cents,
            slot_deleted=slot_deleted,
            slot_id=slot_id,
            slot_name=slot_name,
            status=status,
            submitted_at=submitted_at,
            closed_at=closed_at,
            declined_reason=declined_reason,
            held_by=held_by,
            held_until=held_until,
            paid_on=paid_on,
            paid_source=paid_source,
            payment_due_at=payment_due_at,
            sponsor=sponsor,
        )

        return sponsor_request_item

